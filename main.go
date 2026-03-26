package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kayson1999/MyUserCenter/config"
	"github.com/kayson1999/MyUserCenter/database"
	"github.com/kayson1999/MyUserCenter/handler"
	"github.com/kayson1999/MyUserCenter/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	config.Load()

	// 初始化数据库
	database.Init()

	// 定时清理过期 Token 黑名单（每小时）
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("清理过期 Token 失败: %v", r)
					}
				}()
				database.CleanExpiredTokens()
			}()
		}
	}()

	// 创建 Gin 引擎
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// ── 全局中间件 ──
	r.Use(gin.Recovery())
	r.Use(middleware.Cors())
	r.Use(middleware.Logger())
	r.Use(middleware.APILimiter())

	// ── 认证路由 /auth ──
	auth := r.Group("/auth")
	{
		auth.POST("/register", middleware.AuthLimiter(), middleware.RequireTenant(), handler.Register)
		auth.POST("/login", middleware.AuthLimiter(), middleware.RequireTenant(), handler.Login)
		auth.GET("/verify", middleware.RequireAuth(), middleware.OptionalTenant(), handler.Verify)
		auth.POST("/logout", middleware.RequireAuth(), handler.Logout)
		auth.POST("/refresh", middleware.RequireAuth(), handler.Refresh)
	}

	// ── 用户路由 /user ──
	user := r.Group("/user")
	{
		user.GET("/profile", middleware.RequireAuth(), middleware.OptionalTenant(), handler.GetProfile)
		user.PUT("/profile", middleware.RequireAuth(), handler.UpdateProfile)
		user.PUT("/password", middleware.RequireAuth(), handler.ChangePassword)
	}

	// ── 租户路由 /tenant ──
	tenant := r.Group("/tenant")
	{
		tenant.POST("/register", middleware.RequireInternal(), handler.RegisterTenant)
		tenant.GET("/list", middleware.RequireInternal(), handler.ListTenants)
		tenant.GET("/:appId/secret", middleware.RequireInternal(), handler.GetTenantSecret)
		tenant.PUT("/:appId/status", middleware.RequireInternal(), handler.UpdateTenantStatus)
		tenant.PUT("/:appId/user/:userId/role", middleware.RequireInternal(), handler.UpdateUserRole)
		tenant.PUT("/:appId/user/:userId/extra", middleware.RequireInternal(), handler.UpdateUserExtra)
		tenant.GET("/:appId/users", middleware.RequireInternal(), handler.ListTenantUsers)
	}

	// ── 健康检查路由 /health ──
	r.GET("/health", handler.HealthCheck)
	r.GET("/health/stats", handler.HealthStats)

	// ── 404 处理 ──
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "接口不存在"})
	})

	// ── 启动服务 ──
	addr := fmt.Sprintf(":%d", config.C.Port)
	fmt.Printf("\n🚀 MyUserCenter 用户中心已启动\n")
	fmt.Printf("   地址: http://localhost:%d\n", config.C.Port)
	fmt.Printf("   健康检查: http://localhost:%d/health\n", config.C.Port)
	fmt.Printf("   接口前缀: /auth, /user, /tenant\n\n")

	if err := r.Run(addr); err != nil {
		log.Fatalf("❌ 服务启动失败: %v", err)
	}
}
