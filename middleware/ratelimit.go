package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter 基于 IP 的限流器
type ipLimiter struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPLimiter(r rate.Limit, burst int) *ipLimiter {
	l := &ipLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    burst,
	}
	// 定期清理过期的限流器
	go l.cleanup()
	return l
}

func (l *ipLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry, exists := l.limiters[ip]; exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	limiter := rate.NewLimiter(l.rate, l.burst)
	l.limiters[ip] = &rateLimiterEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	return limiter
}

func (l *ipLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		l.mu.Lock()
		for ip, entry := range l.limiters {
			if time.Since(entry.lastSeen) > 30*time.Minute {
				delete(l.limiters, ip)
			}
		}
		l.mu.Unlock()
	}
}

// APILimiter 通用 API 限流：每秒约 1.67 次（每分钟 100 次），突发 10
func APILimiter() gin.HandlerFunc {
	limiter := newIPLimiter(rate.Limit(100.0/60.0), 10)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.getLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			return
		}
		c.Next()
	}
}

// AuthLimiter 认证接口限流：每 15 分钟 20 次 → 约每秒 0.022 次，突发 5
func AuthLimiter() gin.HandlerFunc {
	limiter := newIPLimiter(rate.Limit(20.0/900.0), 5)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.getLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请 15 分钟后再试"})
			return
		}
		c.Next()
	}
}
