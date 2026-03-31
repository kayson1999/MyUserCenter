package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/kayson1999/MyUserCenter/database"
	"github.com/kayson1999/MyUserCenter/middleware"
	"github.com/kayson1999/MyUserCenter/model"
	"github.com/kayson1999/MyUserCenter/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 随机头像列表
var avatars = []string{"😎", "🚀", "🔥", "💻", "🎮", "⚡", "🌟", "🦊", "🐱", "🐶", "🦁", "🐼", "🐨", "🐯", "🦄", "🐸"}

var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// Register 注册
// POST /auth/register
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	tenant := middleware.GetTenant(c)

	// 参数校验
	if req.Username == "" || req.Password == "" || req.Nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名、密码和昵称不能为空"})
		return
	}
	if !usernameRegex.MatchString(req.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名仅允许字母、数字和下划线"})
		return
	}
	if len(req.Username) < 2 || len(req.Username) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度需在 2-20 之间"})
		return
	}
	if len(req.Password) < 6 || len(req.Password) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度需在 6-30 之间"})
		return
	}
	if len([]rune(req.Nickname)) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "昵称长度不能超过 20"})
		return
	}

	db := database.DB

	// 检查用户名是否已存在
	var existing model.User
	if err := db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已被注册"})
		return
	}

	// 创建用户
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	avatar := avatars[rng.Intn(len(avatars))]

	user := model.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Nickname:     req.Nickname,
		Avatar:       avatar,
	}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	// 建立租户关联
	tu := model.TenantUser{
		TenantID: tenant.ID,
		UserID:   user.ID,
		Role:     "user",
	}
	db.Where("tenant_id = ? AND user_id = ?", tenant.ID, user.ID).FirstOrCreate(&tu)

	// 签发 Token
	token, err := util.SignToken(user.ID, user.Username, tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	// 记录日志
	logAction(db, user.ID, &tenant.ID, "register", c)

	resp := user.ToResponse()
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":         strconv.FormatInt(resp.ID, 10),
			"username":   resp.Username,
			"nickname":   resp.Nickname,
			"avatar":     resp.Avatar,
			"email":      resp.Email,
			"phone":      resp.Phone,
			"status":     resp.Status,
			"created_at": resp.CreatedAt,
			"role":       tu.Role,
			"extra_data": safeParseJSON(tu.ExtraData),
		},
	})
}

// Login 登录
// POST /auth/login
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	tenant := middleware.GetTenant(c)

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}

	db := database.DB

	var user model.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 检查用户全局状态
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用，请联系管理员"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 自动建立租户关联
	tu := model.TenantUser{
		TenantID: tenant.ID,
		UserID:   user.ID,
		Role:     "user",
	}
	db.Where("tenant_id = ? AND user_id = ?", tenant.ID, user.ID).FirstOrCreate(&tu)
	if tu.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "您在该服务中的账号已被禁用"})
		return
	}

	// 签发 Token
	token, err := util.SignToken(user.ID, user.Username, tenant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败，请稍后重试"})
		return
	}

	// 记录日志
	logAction(db, user.ID, &tenant.ID, "login", c)

	resp := user.ToResponse()
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":         strconv.FormatInt(resp.ID, 10),
			"username":   resp.Username,
			"nickname":   resp.Nickname,
			"avatar":     resp.Avatar,
			"email":      resp.Email,
			"phone":      resp.Phone,
			"status":     resp.Status,
			"created_at": resp.CreatedAt,
			"updated_at": resp.UpdatedAt,
			"role":       tu.Role,
			"extra_data": safeParseJSON(tu.ExtraData),
		},
	})
}

// Verify 验证 Token
// GET /auth/verify
func Verify(c *gin.Context) {
	claims := middleware.GetUser(c)
	tenant := middleware.GetTenant(c)

	db := database.DB
	var user model.User
	if err := db.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用"})
		return
	}

	result := gin.H{
		"id":         strconv.FormatInt(user.ID, 10),
		"username":   user.Username,
		"nickname":   user.Nickname,
		"avatar":     user.Avatar,
		"email":      user.Email,
		"phone":      user.Phone,
		"status":     user.Status,
		"created_at": user.CreatedAt,
	}

	if tenant != nil {
		var tu model.TenantUser
		if err := db.Where("tenant_id = ? AND user_id = ?", tenant.ID, user.ID).First(&tu).Error; err == nil {
			result["role"] = tu.Role
			result["tenant_status"] = tu.Status
			result["extra_data"] = safeParseJSON(tu.ExtraData)
		}
	}

	c.JSON(http.StatusOK, gin.H{"user": result})
}

// Logout 登出
// POST /auth/logout
func Logout(c *gin.Context) {
	claims := middleware.GetUser(c)
	tokenStr := middleware.GetToken(c)

	db := database.DB
	tokenHash := sha256Hex(tokenStr)

	// 获取 Token 过期时间
	tokenClaims, err := util.VerifyToken(tokenStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登出失败"})
		return
	}

	blacklist := model.TokenBlacklist{
		TokenHash: tokenHash,
		UserID:    claims.UserID,
		ExpiresAt: tokenClaims.ExpiresAt.Time,
	}
	db.Where("token_hash = ?", tokenHash).FirstOrCreate(&blacklist)

	// 记录日志
	tenantID := &claims.TenantID
	if claims.TenantID == 0 {
		tenantID = nil
	}
	logAction(db, claims.UserID, tenantID, "logout", c)

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "已成功登出"})
}

// Refresh 刷新 Token
// POST /auth/refresh
func Refresh(c *gin.Context) {
	claims := middleware.GetUser(c)
	tokenStr := middleware.GetToken(c)

	db := database.DB

	var user model.User
	if err := db.First(&user, claims.UserID).Error; err != nil || user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号不可用"})
		return
	}

	// 将旧 Token 加入黑名单
	oldTokenHash := sha256Hex(tokenStr)
	oldClaims, _ := util.VerifyToken(tokenStr)
	if oldClaims != nil {
		blacklist := model.TokenBlacklist{
			TokenHash: oldTokenHash,
			UserID:    claims.UserID,
			ExpiresAt: oldClaims.ExpiresAt.Time,
		}
		db.Where("token_hash = ?", oldTokenHash).FirstOrCreate(&blacklist)
	}

	// 签发新 Token
	newToken, err := util.SignToken(user.ID, user.Username, claims.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刷新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newToken})
}

// ── 工具函数 ──

func logAction(db *gorm.DB, userID int64, tenantID *uint, action string, c *gin.Context) {
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	if len(ua) > 200 {
		ua = ua[:200]
	}
	log := model.LoginLog{
		UserID:    userID,
		TenantID:  tenantID,
		Action:    action,
		IP:        ip,
		UserAgent: ua,
	}
	db.Create(&log)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func safeParseJSON(s string) interface{} {
	var result interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return map[string]interface{}{}
	}
	return result
}
