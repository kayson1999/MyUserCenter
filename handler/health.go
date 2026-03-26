package handler

import (
	"net/http"
	"time"

	"github.com/kayson1999/MyUserCenter/database"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查
// GET /health
func HealthCheck(c *gin.Context) {
	status := gin.H{
		"status":  "ok",
		"service": "MyUserCenter",
		"time":    time.Now().Format(time.RFC3339),
	}

	// 检查数据库连接
	sqlDB, err := database.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		status["db"] = "error"
		status["status"] = "degraded"
	} else {
		status["db"] = "ok"
	}

	c.JSON(http.StatusOK, status)
}

// HealthStats 基础统计
// GET /health/stats
func HealthStats(c *gin.Context) {
	db := database.DB

	var userCount, tenantCount, tenantUserCount, todayLogins int64

	db.Table("users").Count(&userCount)
	db.Table("tenants").Count(&tenantCount)
	db.Table("tenant_users").Count(&tenantUserCount)

	today := time.Now().Format("2006-01-02")
	db.Table("login_logs").
		Where("action = ? AND created_at >= ?", "login", today).
		Count(&todayLogins)

	c.JSON(http.StatusOK, gin.H{
		"users":        userCount,
		"tenants":      tenantCount,
		"tenant_users": tenantUserCount,
		"today_logins": todayLogins,
	})
}
