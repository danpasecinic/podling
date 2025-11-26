package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type AuthHandlers struct {
	authStore  AuthStore
	jwtManager *JWTManager
	keyManager *APIKeyManager
	config     Config
}

func NewAuthHandlers(
	authStore AuthStore, jwtManager *JWTManager, keyManager *APIKeyManager, config Config,
) *AuthHandlers {
	return &AuthHandlers{
		authStore:  authStore,
		jwtManager: jwtManager,
		keyManager: keyManager,
		config:     config,
	}
}

func (h *AuthHandlers) RegisterRoutes(e *echo.Echo, authMiddleware *Middleware) {
	auth := e.Group("/api/v1/auth")

	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.RefreshToken)
	auth.POST("/signup", h.Signup)

	protected := auth.Group("")
	protected.Use(authMiddleware.Authenticate())

	protected.GET("/me", h.GetCurrentUser)
	protected.POST("/api-keys", h.CreateAPIKey, authMiddleware.RequireRole(RoleAdmin))
	protected.GET("/api-keys", h.ListAPIKeys, authMiddleware.RequireRole(RoleAdmin))
	protected.DELETE("/api-keys/:id", h.RevokeAPIKey, authMiddleware.RequireRole(RoleAdmin))

	protected.POST("/users", h.CreateUser, authMiddleware.RequireRole(RoleAdmin))
	protected.GET("/users", h.ListUsers, authMiddleware.RequireRole(RoleAdmin))
	protected.GET("/users/:id", h.GetUser, authMiddleware.RequireRole(RoleAdmin))
	protected.PUT("/users/:id", h.UpdateUser, authMiddleware.RequireRole(RoleAdmin))
	protected.DELETE("/users/:id", h.DeleteUser, authMiddleware.RequireRole(RoleAdmin))
}

func (h *AuthHandlers) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username and password are required"})
	}

	user, err := h.authStore.GetUserByUsername(req.Username)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	if user.Disabled {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "account disabled"})
	}

	if !VerifyPassword(req.Password, user.PasswordHash) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	token, expiresAt, err := h.jwtManager.GenerateToken(&user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	refreshToken, _, err := h.jwtManager.GenerateRefreshToken(&user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate refresh token"})
	}

	now := time.Now()
	_ = h.authStore.UpdateUser(user.ID, UserUpdate{LastLogin: &now})

	return c.JSON(
		http.StatusOK, LoginResponse{
			Token:        token,
			RefreshToken: refreshToken,
			ExpiresAt:    expiresAt,
			User: UserInfo{
				ID:       user.ID,
				Username: user.Username,
				Role:     user.Role,
			},
		},
	)
}

type SignupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandlers) Signup(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username and password are required"})
	}

	if len(req.Username) < 3 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username must be at least 3 characters"})
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	now := time.Now()
	user := User{
		ID:           fmt.Sprintf("user-%s-%s", now.Format("20060102150405"), randomString(8)),
		Username:     req.Username,
		PasswordHash: passwordHash,
		Role:         RoleViewer,
		CreatedAt:    now,
		Disabled:     false,
	}

	if err := h.authStore.AddUser(user); err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "username already exists"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create account"})
	}

	token, expiresAt, err := h.jwtManager.GenerateToken(&user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	refreshToken, _, err := h.jwtManager.GenerateRefreshToken(&user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate refresh token"})
	}

	return c.JSON(
		http.StatusCreated, LoginResponse{
			Token:        token,
			RefreshToken: refreshToken,
			ExpiresAt:    expiresAt,
			User: UserInfo{
				ID:       user.ID,
				Username: user.Username,
				Role:     user.Role,
			},
		},
	)
}

func (h *AuthHandlers) RefreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "refresh token is required"})
	}

	token, expiresAt, err := h.jwtManager.RefreshToken(req.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
	}

	return c.JSON(
		http.StatusOK, map[string]interface{}{
			"token":     token,
			"expiresAt": expiresAt,
		},
	)
}

func (h *AuthHandlers) GetCurrentUser(c echo.Context) error {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}

	return c.JSON(
		http.StatusOK, map[string]interface{}{
			"userId":   authCtx.UserID,
			"username": authCtx.Username,
			"role":     authCtx.Role,
			"authType": authCtx.AuthType,
			"nodeId":   authCtx.NodeID,
		},
	)
}

func (h *AuthHandlers) CreateAPIKey(c echo.Context) error {
	var req CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	if !req.Role.IsValid() {
		req.Role = RoleOperator
	}

	apiKey, plainKey, err := h.keyManager.CreateAPIKey(req.Name, req.NodeID, req.Role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create API key"})
	}

	if err := h.authStore.AddAPIKey(*apiKey); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store API key"})
	}

	return c.JSON(
		http.StatusCreated, CreateAPIKeyResponse{
			Key:    plainKey,
			APIKey: *apiKey,
		},
	)
}

func (h *AuthHandlers) ListAPIKeys(c echo.Context) error {
	keys, err := h.authStore.ListAPIKeys()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list API keys"})
	}

	return c.JSON(http.StatusOK, keys)
}

func (h *AuthHandlers) RevokeAPIKey(c echo.Context) error {
	keyID := c.Param("id")

	if err := h.authStore.RevokeAPIKey(keyID); err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "API key not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to revoke API key"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "API key revoked"})
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

func (h *AuthHandlers) CreateUser(c echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username and password are required"})
	}

	if !req.Role.IsValid() {
		req.Role = RoleViewer
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	now := time.Now()
	user := User{
		ID:           fmt.Sprintf("user-%s-%s", now.Format("20060102150405"), randomString(8)),
		Username:     req.Username,
		PasswordHash: passwordHash,
		Role:         req.Role,
		CreatedAt:    now,
		Disabled:     false,
	}

	if err := h.authStore.AddUser(user); err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "username already exists"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
	}

	user.PasswordHash = ""
	return c.JSON(http.StatusCreated, user)
}

func (h *AuthHandlers) ListUsers(c echo.Context) error {
	users, err := h.authStore.ListUsers()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list users"})
	}

	for i := range users {
		users[i].PasswordHash = ""
	}

	return c.JSON(http.StatusOK, users)
}

func (h *AuthHandlers) GetUser(c echo.Context) error {
	userID := c.Param("id")

	user, err := h.authStore.GetUser(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get user"})
	}

	user.PasswordHash = ""
	return c.JSON(http.StatusOK, user)
}

type UpdateUserRequest struct {
	Password *string `json:"password,omitempty"`
	Role     *Role   `json:"role,omitempty"`
	Disabled *bool   `json:"disabled,omitempty"`
}

func (h *AuthHandlers) UpdateUser(c echo.Context) error {
	userID := c.Param("id")

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	update := UserUpdate{}

	if req.Password != nil {
		passwordHash, err := HashPassword(*req.Password)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		update.PasswordHash = &passwordHash
	}

	if req.Role != nil {
		if !req.Role.IsValid() {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role"})
		}
		update.Role = req.Role
	}

	if req.Disabled != nil {
		update.Disabled = req.Disabled
	}

	if err := h.authStore.UpdateUser(userID, update); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update user"})
	}

	user, _ := h.authStore.GetUser(userID)
	user.PasswordHash = ""
	return c.JSON(http.StatusOK, user)
}

func (h *AuthHandlers) DeleteUser(c echo.Context) error {
	userID := c.Param("id")

	authCtx := GetAuthContext(c)
	if authCtx != nil && authCtx.UserID == userID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot delete your own account"})
	}

	if err := h.authStore.DeleteUser(userID); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete user"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "user deleted"})
}
