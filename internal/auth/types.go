package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleViewer:
		return true
	}
	return false
}

func (r Role) CanWrite() bool {
	return r == RoleAdmin || r == RoleOperator
}

func (r Role) CanDelete() bool {
	return r == RoleAdmin
}

func (r Role) CanManageNodes() bool {
	return r == RoleAdmin
}

type APIKey struct {
	ID        string     `json:"id"`
	KeyHash   string     `json:"-"`
	KeyPrefix string     `json:"keyPrefix"`
	Name      string     `json:"name"`
	NodeID    string     `json:"nodeId,omitempty"`
	Role      Role       `json:"role"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	LastUsed  *time.Time `json:"lastUsed,omitempty"`
	Revoked   bool       `json:"revoked"`
}

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	Role         Role       `json:"role"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastLogin    *time.Time `json:"lastLogin,omitempty"`
	Disabled     bool       `json:"disabled"`
}

type Claims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

type AuthContext struct {
	UserID   string
	Username string
	Role     Role
	NodeID   string
	AuthType string
}

type Config struct {
	Enabled        bool
	JWTSecret      string
	APIKeySecret   string
	TokenExpiry    time.Duration
	RefreshExpiry  time.Duration
	AllowedOrigins []string
}

func DefaultConfig() Config {
	return Config{
		Enabled:       false,
		TokenExpiry:   24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt"`
	User         UserInfo  `json:"user"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

type CreateAPIKeyRequest struct {
	Name   string `json:"name"`
	NodeID string `json:"nodeId,omitempty"`
	Role   Role   `json:"role"`
}

type CreateAPIKeyResponse struct {
	Key    string `json:"key"`
	APIKey APIKey `json:"apiKey"`
}
