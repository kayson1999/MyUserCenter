package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 全局配置
type Config struct {
	Port           int
	JWTSecret      string
	JWTExpiresIn   time.Duration
	InternalSecret string

	// 数据库
	DBType     string // mysql 或 sqlite
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBPath     string // SQLite 文件路径
}

var C Config

// Load 加载配置
func Load() {
	// 加载 .env 文件（忽略错误）
	_ = godotenv.Load()

	C = Config{
		Port:           getEnvInt("PORT", 4000),
		JWTSecret:      getEnv("JWT_SECRET", "usercenter_secret_mamba_2026"),
		JWTExpiresIn:   parseDuration(getEnv("JWT_EXPIRES_IN", "7d")),
		InternalSecret: getEnv("INTERNAL_SECRET", "internal_shared_secret_mamba_2026"),

		DBType:     strings.ToLower(getEnv("DB_TYPE", "mysql")),
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnvInt("DB_PORT", 3306),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "root"),
		DBName:     getEnv("DB_NAME", "usercenter"),
		DBPath:     getEnv("DB_PATH", "./data/usercenter.db"),
	}
}

// DSN 返回 MySQL 连接字符串
func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + strconv.Itoa(c.DBPort) + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

// IsSQLite 判断是否使用 SQLite
func (c *Config) IsSQLite() bool {
	return c.DBType == "sqlite"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}

// parseDuration 解析类似 "7d", "24h", "30m" 的时间字符串
func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 7 * 24 * time.Hour // 默认 7 天
	}
	return d
}
