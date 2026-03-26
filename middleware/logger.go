package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// 健康检查接口不记录日志
		if c.Request.URL.Path == "/health" {
			return
		}

		duration := time.Since(start)
		log.Printf("[%s] %s → %d (%v)",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
		)
	}
}
