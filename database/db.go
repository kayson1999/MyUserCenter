package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kayson1999/MyUserCenter/config"
	"github.com/kayson1999/MyUserCenter/model"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库连接、建表、种子数据
func Init() {
	var err error
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	if config.C.IsSQLite() {
		// SQLite 模式
		dbPath := config.C.DBPath

		// 确保目录存在
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("❌ 创建 SQLite 数据目录失败: %v", err)
		}

		DB, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
		if err != nil {
			log.Fatalf("❌ SQLite 数据库连接失败: %v", err)
		}

		// SQLite 性能优化
		sqlDB, _ := DB.DB()
		sqlDB.SetMaxOpenConns(1) // SQLite 建议单连接
		sqlDB.SetMaxIdleConns(1)

		// 开启 WAL 模式和外键支持
		DB.Exec("PRAGMA journal_mode=WAL")
		DB.Exec("PRAGMA foreign_keys=ON")

		log.Printf("📦 使用 SQLite 数据库: %s", dbPath)
	} else {
		// MySQL 模式
		DB, err = gorm.Open(mysql.Open(config.C.DSN()), gormConfig)
		if err != nil {
			log.Fatalf("❌ MySQL 数据库连接失败: %v", err)
		}

		// 连接池配置
		sqlDB, _ := DB.DB()
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)

		log.Println("📦 使用 MySQL 数据库")
	}

	// 自动建表
	if err := DB.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.TenantUser{},
		&model.TokenBlacklist{},
		&model.LoginLog{},
	); err != nil {
		log.Fatalf("❌ 数据库建表失败: %v", err)
	}

	// 种子数据
	seedTenants()

	log.Println("✅ 数据库初始化完成")
}

// seedTenants 初始化种子租户
func seedTenants() {
	tenants := []model.Tenant{
		{
			AppID:          "myhomepage",
			AppSecret:      generateSecret(),
			Name:           "MyHomePage",
			Description:    "个人主页 — 打工人打卡等服务",
			AllowedOrigins: `["http://localhost:3000","http://localhost:3001"]`,
		},
		{
			AppID:          "decisioner",
			AppSecret:      generateSecret(),
			Name:           "冲动是魔鬼",
			Description:    "决策分析助手",
			AllowedOrigins: `["http://localhost:9090"]`,
		},
	}

	for _, t := range tenants {
		// 仅在不存在时插入
		var count int64
		DB.Model(&model.Tenant{}).Where("app_id = ?", t.AppID).Count(&count)
		if count == 0 {
			DB.Create(&t)
		}
	}
}

func generateSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CleanExpiredTokens 清理过期的 Token 黑名单
func CleanExpiredTokens() {
	result := DB.Where("expires_at < ?", time.Now()).Delete(&model.TokenBlacklist{})
	if result.RowsAffected > 0 {
		fmt.Printf("🧹 已清理 %d 条过期 Token\n", result.RowsAffected)
	}
}
