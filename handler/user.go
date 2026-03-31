package handler

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/kayson1999/MyUserCenter/database"
	"github.com/kayson1999/MyUserCenter/middleware"
	"github.com/kayson1999/MyUserCenter/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// GetProfile 获取当前用户信息
// GET /user/profile
func GetProfile(c *gin.Context) {
	claims := middleware.GetUser(c)
	tenant := middleware.GetTenant(c)

	db := database.DB
	var user model.User
	if err := db.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
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
		"updated_at": user.UpdatedAt,
	}

	// 如果指定了租户，附加租户下的信息
	if tenant != nil {
		var tu model.TenantUser
		if err := db.Where("tenant_id = ? AND user_id = ?", tenant.ID, user.ID).First(&tu).Error; err == nil {
			result["role"] = tu.Role
			result["tenant_status"] = tu.Status
			result["extra_data"] = safeParseJSON(tu.ExtraData)
		}
	}

	// 获取用户关联的所有租户
	type TenantInfo struct {
		AppID    string `json:"app_id"`
		Name     string `json:"name"`
		Role     string `json:"role"`
		Status   string `json:"status"`
		JoinedAt string `json:"joined_at"`
	}
	var tenants []TenantInfo
	db.Table("tenant_users tu").
		Select("t.app_id, t.name, tu.role, tu.status, tu.created_at as joined_at").
		Joins("JOIN tenants t ON t.id = tu.tenant_id").
		Where("tu.user_id = ?", user.ID).
		Order("tu.created_at").
		Scan(&tenants)

	result["tenants"] = tenants

	c.JSON(http.StatusOK, gin.H{"user": result})
}

// UpdateProfile 更新个人信息
// PUT /user/profile
func UpdateProfile(c *gin.Context) {
	claims := middleware.GetUser(c)

	var req struct {
		Nickname *string `json:"nickname"`
		Avatar   *string `json:"avatar"`
		Email    *string `json:"email"`
		Phone    *string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	db := database.DB
	updates := map[string]interface{}{}

	if req.Nickname != nil {
		if len([]rune(*req.Nickname)) > 20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "昵称长度不能超过 20"})
			return
		}
		updates["nickname"] = *req.Nickname
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Email != nil {
		if *req.Email != "" && !emailRegex.MatchString(*req.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确"})
			return
		}
		updates["email"] = *req.Email
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有需要更新的字段"})
		return
	}

	db.Model(&model.User{}).Where("id = ?", claims.UserID).Updates(updates)

	var user model.User
	db.First(&user, claims.UserID)

	c.JSON(http.StatusOK, gin.H{"user": user.ToResponse()})
}

// ChangePassword 修改密码
// PUT /user/password
func ChangePassword(c *gin.Context) {
	claims := middleware.GetUser(c)

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误"})
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入旧密码和新密码"})
		return
	}
	if len(req.NewPassword) < 6 || len(req.NewPassword) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码长度需在 6-30 之间"})
		return
	}

	db := database.DB
	var user model.User
	if err := db.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "旧密码错误"})
		return
	}

	// 更新密码
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "修改密码失败"})
		return
	}

	db.Model(&model.User{}).Where("id = ?", claims.UserID).Update("password_hash", string(newHash))

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "密码修改成功"})
}
