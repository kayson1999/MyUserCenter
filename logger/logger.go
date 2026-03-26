package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kayson1999/MyUserCenter/config"
)

var (
	// 当前日志文件句柄
	currentFile *os.File
	// 当前日志文件对应的日期（用于判断是否需要滚动）
	currentDate string
	// 互斥锁，保护日志文件切换
	mu sync.Mutex
)

// Init 初始化日志系统
// 根据配置决定日志输出到文件还是控制台
func Init() {
	if !config.C.LogToFile {
		// 仅输出到控制台
		log.SetOutput(os.Stdout)
		log.SetFlags(log.Ldate | log.Ltime)
		return
	}

	// 确保日志目录存在
	if err := os.MkdirAll(config.C.LogDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建日志目录失败: %v，回退到控制台输出\n", err)
		log.SetOutput(os.Stdout)
		log.SetFlags(log.Ldate | log.Ltime)
		return
	}

	// 打开当天日志文件
	if err := rotateIfNeeded(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 打开日志文件失败: %v，回退到控制台输出\n", err)
		log.SetOutput(os.Stdout)
		log.SetFlags(log.Ldate | log.Ltime)
		return
	}

	// 启动日志滚动检查协程（每分钟检查一次是否跨天）
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			_ = rotateIfNeeded()
			mu.Unlock()
		}
	}()
}

// rotateIfNeeded 检查是否需要滚动日志文件（按天）
// 调用方需持有 mu 锁，或在 Init 中首次调用时无需加锁
func rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")
	if today == currentDate && currentFile != nil {
		return nil
	}

	// 关闭旧文件
	if currentFile != nil {
		_ = currentFile.Close()
	}

	// 构建日志文件路径：{LogDir}/{LogFilePrefix}-{date}.log
	filename := fmt.Sprintf("%s-%s.log", config.C.LogFilePrefix, today)
	logPath := filepath.Join(config.C.LogDir, filename)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	currentFile = f
	currentDate = today

	// 同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime)

	// 清理过期日志文件
	go cleanOldLogs()

	return nil
}

// cleanOldLogs 清理超过保留天数的日志文件
func cleanOldLogs() {
	if config.C.LogMaxDays <= 0 {
		return // 不限制保留天数
	}

	cutoff := time.Now().AddDate(0, 0, -config.C.LogMaxDays)
	pattern := filepath.Join(config.C.LogDir, config.C.LogFilePrefix+"-*.log")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
			log.Printf("🧹 已清理过期日志文件: %s", filepath.Base(path))
		}
	}
}

// Close 关闭日志文件（程序退出时调用）
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if currentFile != nil {
		_ = currentFile.Close()
		currentFile = nil
	}
}
