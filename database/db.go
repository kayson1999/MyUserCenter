package database

import (
	"log"
	"encoding/json"
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

	log.Println("✅ 数据库初始化完成")

	// 预注册种子租户
	seedTenants()
}

// CleanExpiredTokens 清理过期的 Token 黑名单
func CleanExpiredTokens() {
	result := DB.Where("expires_at < ?", time.Now()).Delete(&model.TokenBlacklist{})
	if result.RowsAffected > 0 {
		log.Printf("🧹 已清理 %d 条过期 Token", result.RowsAffected)
	}
}

// seedTenants 预注册种子租户
// 从配置中读取预注册租户列表，如果租户不存在则创建，如果已存在则更新 app_secret
func seedTenants() {
	tenants := config.C.SeedTenants
	if len(tenants) == 0 {
		return
	}

	for _, seed := range tenants {
		var existing model.Tenant
		err := DB.Where("app_id = ?", seed.AppID).First(&existing).Error

		if err != nil {
			origins, _ := json.Marshal([]string{})
			tenant := model.Tenant{
				AppID:          seed.AppID,
				AppSecret:      seed.AppSecret,
				Name:           seed.Name,
				AllowedOrigins: string(origins),
			}
			if createErr := DB.Create(&tenant).Error; createErr != nil {
				log.Printf("[seed] 预注册租户 %s 失败: %v", seed.AppID, createErr)
			} else {
				log.Printf("[seed] 预注册租户: %s (%s)", seed.AppID, seed.Name)
			}
		} else {
			if existing.AppSecret != seed.AppSecret {
				DB.Model(&existing).Update("app_secret", seed.AppSecret)
				log.Printf("[seed] 更新租户密钥: %s (%s)", seed.AppID, seed.Name)
			} else {
				log.Printf("[seed] 租户已存在: %s (%s)", seed.AppID, seed.Name)
			}
		}
	}
}
