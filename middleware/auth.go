package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kayson1999/MyUserCenter/config"
	"github.com/kayson1999/MyUserCenter/database"
	"github.com/kayson1999/MyUserCenter/model"
	"github.com/kayson1999/MyUserCenter/util"

	"github.com/gin-gonic/gin"
)

// RequireAuth JWT Token 验证中间件
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供认证 Token"})
			return
		}

		tokenStr := authHeader[7:]

		// 检查 Token 是否在黑名单中
		tokenHash := sha256Hash(tokenStr)
		var count int64
		database.DB.Model(&model.TokenBlacklist{}).Where("token_hash = ?", tokenHash).Count(&count)
		if count > 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token 已失效，请重新登录"})
			return
		}

		// 验证 Token
		claims, err := util.VerifyToken(tokenStr)
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token 已过期，请重新登录"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的 Token"})
			return
		}

		c.Set("user", claims)
		c.Set("token", tokenStr)
		c.Next()
	}
}

// RequireTenant 租户身份验证中间件
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.GetHeader("X-App-Id")
		appSign := c.GetHeader("X-App-Sign")
		timestamp := c.GetHeader("X-Timestamp")

		if appID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "缺少 X-App-Id 请求头"})
			return
		}

		var tenant model.Tenant
		if err := database.DB.Where("app_id = ? AND status = ?", appID, "active").First(&tenant).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无效的租户标识或租户已禁用"})
			return
		}

		// 如果提供了签名，则验证签名
		if appSign != "" && timestamp != "" {
			now := time.Now().Unix()
			ts, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil || math.Abs(float64(now-ts)) > 300 {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "请求时间戳已过期"})
				return
			}

			// 验证 HMAC 签名
			bodyBytes, _ := c.GetRawData()
			bodyStr := string(bodyBytes)
			signPayload := timestamp + appID + bodyStr
			expectedSign := hmacSHA256(tenant.AppSecret, signPayload)

			if appSign != expectedSign {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "签名验证失败"})
				return
			}

			// 恢复 body 以便后续 handler 读取
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Set("tenant", &tenant)
		c.Next()
	}
}

// OptionalTenant 可选的租户验证中间件
func OptionalTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.GetHeader("X-App-Id")
		if appID != "" {
			RequireTenant()(c)
			return
		}
		c.Next()
	}
}

// RequireInternal 内部接口认证中间件
func RequireInternal() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := c.GetHeader("X-Internal-Secret")
		if secret == "" || secret != config.C.InternalSecret {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权访问内部接口"})
			return
		}
		c.Next()
	}
}

// GetUser 从上下文获取用户信息
func GetUser(c *gin.Context) *util.Claims {
	if val, exists := c.Get("user"); exists {
		if claims, ok := val.(*util.Claims); ok {
			return claims
		}
	}
	return nil
}

// GetTenant 从上下文获取租户信息
func GetTenant(c *gin.Context) *model.Tenant {
	if val, exists := c.Get("tenant"); exists {
		if tenant, ok := val.(*model.Tenant); ok {
			return tenant
		}
	}
	return nil
}

// GetToken 从上下文获取原始 Token
func GetToken(c *gin.Context) string {
	if val, exists := c.Get("token"); exists {
		if token, ok := val.(string); ok {
			return token
		}
	}
	return ""
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func hmacSHA256(secret, data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
