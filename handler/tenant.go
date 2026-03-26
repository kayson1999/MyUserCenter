package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/kayson1999/MyUserCenter/database"
	"github.com/kayson1999/MyUserCenter/model"

	"github.com/gin-gonic/gin"
)

var appIDRegex = regexp.MustCompile(`^[a-z0-9_-]+$`)

// RegisterTenant 注册新租户
// POST /tenant/register
func RegisterTenant(c *gin.Context) {
	var req struct {
		AppID          string   `json:"app_id"`
		Name           string   `json:"name"`
		Description    string   `json:"description"`
		AllowedOrigins []string `json:"allowed_origins"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	if req.AppID == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id 和 name 不能为空"})
		return
	}
	if !appIDRegex.MatchString(req.AppID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id 仅允许小写字母、数字、下划线和连字符"})
		return
	}

	db := database.DB

	// 检查 app_id 是否已存在
	var count int64
	db.Model(&model.Tenant{}).Where("app_id = ?", req.AppID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该 app_id 已被注册"})
		return
	}

	// 生成 app_secret
	secretBytes := make([]byte, 32)
	_, _ = rand.Read(secretBytes)
	appSecret := hex.EncodeToString(secretBytes)

	origins := req.AllowedOrigins
	if origins == nil {
		origins = []string{}
	}
	originsJSON, _ := json.Marshal(origins)

	tenant := model.Tenant{
		AppID:          req.AppID,
		AppSecret:      appSecret,
		Name:           req.Name,
		Description:    req.Description,
		AllowedOrigins: string(originsJSON),
	}
	if err := db.Create(&tenant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册租户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"tenant": gin.H{
			"id":              tenant.ID,
			"app_id":          tenant.AppID,
			"app_secret":      tenant.AppSecret,
			"name":            tenant.Name,
			"description":     tenant.Description,
			"allowed_origins": origins,
		},
	})
}

// ListTenants 查询所有租户
// GET /tenant/list
func ListTenants(c *gin.Context) {
	db := database.DB
	var tenants []model.Tenant
	db.Order("id").Find(&tenants)

	var result []gin.H
	for _, t := range tenants {
		var count int64
		db.Model(&model.TenantUser{}).Where("tenant_id = ? AND status = ?", t.ID, "active").Count(&count)

		result = append(result, gin.H{
			"id":              t.ID,
			"app_id":          t.AppID,
			"name":            t.Name,
			"description":     t.Description,
			"allowed_origins": safeParseJSON(t.AllowedOrigins),
			"status":          t.Status,
			"created_at":      t.CreatedAt,
			"user_count":      count,
		})
	}

	c.JSON(http.StatusOK, gin.H{"tenants": result})
}

// GetTenantSecret 查询租户密钥
// GET /tenant/:appId/secret
func GetTenantSecret(c *gin.Context) {
	appID := c.Param("appId")
	db := database.DB

	var tenant model.Tenant
	if err := db.Where("app_id = ?", appID).First(&tenant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "租户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant": gin.H{
			"app_id":     tenant.AppID,
			"app_secret": tenant.AppSecret,
			"name":       tenant.Name,
		},
	})
}

// UpdateTenantStatus 启用/禁用租户
// PUT /tenant/:appId/status
func UpdateTenantStatus(c *gin.Context) {
	appID := c.Param("appId")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	if req.Status != "active" && req.Status != "disabled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status 仅允许 active 或 disabled"})
		return
	}

	db := database.DB
	result := db.Model(&model.Tenant{}).Where("app_id = ?", appID).Update("status", req.Status)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "租户不存在"})
		return
	}

	msg := "租户已启用"
	if req.Status == "disabled" {
		msg = "租户已禁用"
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": msg})
}

// UpdateUserRole 设置用户在租户下的角色
// PUT /tenant/:appId/user/:userId/role
func UpdateUserRole(c *gin.Context) {
	appID := c.Param("appId")
	userIDStr := c.Param("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户 ID"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	if req.Role != "user" && req.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role 仅允许 user 或 admin"})
		return
	}

	db := database.DB
	var tenant model.Tenant
	if err := db.Where("app_id = ?", appID).First(&tenant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "租户不存在"})
		return
	}

	result := db.Model(&model.TenantUser{}).
		Where("tenant_id = ? AND user_id = ?", tenant.ID, uint(userID)).
		Update("role", req.Role)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户未关联该租户"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "用户角色已更新为 " + req.Role})
}

// UpdateUserExtra 更新用户在租户下的扩展数据
// PUT /tenant/:appId/user/:userId/extra
func UpdateUserExtra(c *gin.Context) {
	appID := c.Param("appId")
	userIDStr := c.Param("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户 ID"})
		return
	}

	var req struct {
		ExtraData interface{} `json:"extra_data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	if req.ExtraData == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "extra_data 必须是 JSON 对象"})
		return
	}

	extraJSON, err := json.Marshal(req.ExtraData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "extra_data 必须是 JSON 对象"})
		return
	}

	db := database.DB
	var tenant model.Tenant
	if err := db.Where("app_id = ?", appID).First(&tenant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "租户不存在"})
		return
	}

	result := db.Model(&model.TenantUser{}).
		Where("tenant_id = ? AND user_id = ?", tenant.ID, uint(userID)).
		Update("extra_data", string(extraJSON))
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户未关联该租户"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListTenantUsers 查询租户下的用户列表
// GET /tenant/:appId/users
func ListTenantUsers(c *gin.Context) {
	appID := c.Param("appId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if pageSize > 50 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	db := database.DB
	var tenant model.Tenant
	if err := db.Where("app_id = ?", appID).First(&tenant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "租户不存在"})
		return
	}

	var total int64
	db.Model(&model.TenantUser{}).Where("tenant_id = ?", tenant.ID).Count(&total)

	type UserItem struct {
		ID           uint   `json:"id"`
		Username     string `json:"username"`
		Nickname     string `json:"nickname"`
		Avatar       string `json:"avatar"`
		UserStatus   string `json:"user_status"`
		Role         string `json:"role"`
		TenantStatus string `json:"tenant_status"`
		ExtraData    string `json:"extra_data"`
		JoinedAt     string `json:"joined_at"`
	}
	var users []UserItem
	db.Table("tenant_users tu").
		Select("u.id, u.username, u.nickname, u.avatar, u.status as user_status, tu.role, tu.status as tenant_status, tu.extra_data, tu.created_at as joined_at").
		Joins("JOIN users u ON u.id = tu.user_id").
		Where("tu.tenant_id = ?", tenant.ID).
		Order("tu.created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(&users)

	// 解析 extra_data
	var result []gin.H
	for _, u := range users {
		result = append(result, gin.H{
			"id":            u.ID,
			"username":      u.Username,
			"nickname":      u.Nickname,
			"avatar":        u.Avatar,
			"user_status":   u.UserStatus,
			"role":          u.Role,
			"tenant_status": u.TenantStatus,
			"extra_data":    safeParseJSON(u.ExtraData),
			"joined_at":     u.JoinedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"users":     result,
	})
}
