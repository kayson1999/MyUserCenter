package model

import "time"

// Tenant 租户（接入的服务）
type Tenant struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID          string    `gorm:"column:app_id;uniqueIndex;size:64;not null" json:"app_id"`
	AppSecret      string    `gorm:"column:app_secret;size:128;not null" json:"app_secret,omitempty"`
	Name           string    `gorm:"size:100;not null" json:"name"`
	Description    string    `gorm:"size:500;default:''" json:"description"`
	AllowedOrigins string    `gorm:"column:allowed_origins;type:text;default:'[]'" json:"allowed_origins"`
	Status         string    `gorm:"size:20;default:active" json:"status"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Tenant) TableName() string { return "tenants" }

// User 全局用户表
type User struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	PasswordHash string    `gorm:"column:password_hash;size:128;not null" json:"-"`
	Nickname     string    `gorm:"size:50;not null" json:"nickname"`
	Avatar       string    `gorm:"size:50;default:'😎'" json:"avatar"`
	Email        string    `gorm:"size:100;default:''" json:"email"`
	Phone        string    `gorm:"size:20;default:''" json:"phone"`
	Status       string    `gorm:"size:20;default:active" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "users" }

// TenantUser 租户-用户关联表
type TenantUser struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint      `gorm:"column:tenant_id;not null;uniqueIndex:idx_tenant_user" json:"tenant_id"`
	UserID    uint      `gorm:"column:user_id;not null;uniqueIndex:idx_tenant_user" json:"user_id"`
	Role      string    `gorm:"size:20;default:user" json:"role"`
	ExtraData string    `gorm:"column:extra_data;type:text;default:'{}'" json:"extra_data"`
	Status    string    `gorm:"size:20;default:active" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	User   User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (TenantUser) TableName() string { return "tenant_users" }

// TokenBlacklist Token 黑名单
type TokenBlacklist struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TokenHash string    `gorm:"column:token_hash;uniqueIndex;size:64;not null" json:"token_hash"`
	UserID    uint      `gorm:"column:user_id;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null;index" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (TokenBlacklist) TableName() string { return "token_blacklist" }

// LoginLog 登录日志
type LoginLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"column:user_id;not null;index:idx_login_logs_user" json:"user_id"`
	TenantID  *uint     `gorm:"column:tenant_id;index:idx_login_logs_tenant" json:"tenant_id"`
	Action    string    `gorm:"size:20;not null" json:"action"`
	IP        string    `gorm:"size:100;default:''" json:"ip"`
	UserAgent string    `gorm:"column:user_agent;size:200;default:''" json:"user_agent"`
	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_login_logs_user;index:idx_login_logs_tenant" json:"created_at"`
}

func (LoginLog) TableName() string { return "login_logs" }

// ── 用于 JSON 响应的辅助结构 ──

// UserResponse 用户信息响应（不含密码）
type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// ToResponse 将 User 转为安全响应
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Email:     u.Email,
		Phone:     u.Phone,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
