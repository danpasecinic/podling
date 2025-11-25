package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		role      Role
		canWrite  bool
		canDelete bool
		canManage bool
	}{
		{RoleAdmin, true, true, true},
		{RoleOperator, true, false, false},
		{RoleViewer, false, false, false},
	}

	for _, tt := range tests {
		t.Run(
			string(tt.role), func(t *testing.T) {
				if tt.role.CanWrite() != tt.canWrite {
					t.Errorf("CanWrite() = %v, want %v", tt.role.CanWrite(), tt.canWrite)
				}
				if tt.role.CanDelete() != tt.canDelete {
					t.Errorf("CanDelete() = %v, want %v", tt.role.CanDelete(), tt.canDelete)
				}
				if tt.role.CanManageNodes() != tt.canManage {
					t.Errorf("CanManageNodes() = %v, want %v", tt.role.CanManageNodes(), tt.canManage)
				}
			},
		)
	}
}

func TestRoleIsValid(t *testing.T) {
	tests := []struct {
		role  Role
		valid bool
	}{
		{RoleAdmin, true},
		{RoleOperator, true},
		{RoleViewer, true},
		{Role("invalid"), false},
		{Role(""), false},
	}

	for _, tt := range tests {
		t.Run(
			string(tt.role), func(t *testing.T) {
				if tt.role.IsValid() != tt.valid {
					t.Errorf("IsValid() = %v, want %v", tt.role.IsValid(), tt.valid)
				}
			},
		)
	}
}

func TestJWTManager(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, 24*time.Hour)

	user := &User{
		ID:       "user-123",
		Username: "testuser",
		Role:     RoleAdmin,
	}

	t.Run(
		"GenerateToken", func(t *testing.T) {
			token, expiresAt, err := manager.GenerateToken(user)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}
			if token == "" {
				t.Error("GenerateToken() returned empty token")
			}
			if expiresAt.Before(time.Now()) {
				t.Error("GenerateToken() returned expired time")
			}
		},
	)

	t.Run(
		"ValidateToken", func(t *testing.T) {
			token, _, _ := manager.GenerateToken(user)

			claims, err := manager.ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken() error = %v", err)
			}
			if claims.UserID != user.ID {
				t.Errorf("claims.UserID = %v, want %v", claims.UserID, user.ID)
			}
			if claims.Username != user.Username {
				t.Errorf("claims.Username = %v, want %v", claims.Username, user.Username)
			}
			if claims.Role != user.Role {
				t.Errorf("claims.Role = %v, want %v", claims.Role, user.Role)
			}
		},
	)

	t.Run(
		"ValidateToken_Invalid", func(t *testing.T) {
			_, err := manager.ValidateToken("invalid-token")
			if err == nil {
				t.Error("ValidateToken() should fail for invalid token")
			}
		},
	)

	t.Run(
		"RefreshToken", func(t *testing.T) {
			refreshToken, _, _ := manager.GenerateRefreshToken(user)

			newToken, _, err := manager.RefreshToken(refreshToken)
			if err != nil {
				t.Fatalf("RefreshToken() error = %v", err)
			}
			if newToken == "" {
				t.Error("RefreshToken() returned empty token")
			}
		},
	)

	t.Run(
		"RefreshToken_WithAccessToken", func(t *testing.T) {
			accessToken, _, _ := manager.GenerateToken(user)

			_, _, err := manager.RefreshToken(accessToken)
			if err == nil {
				t.Error("RefreshToken() should fail with access token")
			}
		},
	)
}

func TestAPIKeyManager(t *testing.T) {
	manager := NewAPIKeyManager("test-secret")

	t.Run(
		"GenerateKey", func(t *testing.T) {
			plainKey, keyHash, keyPrefix, err := manager.GenerateKey()
			if err != nil {
				t.Fatalf("GenerateKey() error = %v", err)
			}
			if !strings.HasPrefix(plainKey, "plk_") {
				t.Errorf("plainKey should start with 'plk_', got %v", plainKey[:10])
			}
			if keyHash == "" {
				t.Error("keyHash should not be empty")
			}
			if keyPrefix == "" {
				t.Error("keyPrefix should not be empty")
			}
		},
	)

	t.Run(
		"ValidateKeyFormat", func(t *testing.T) {
			plainKey, _, _, _ := manager.GenerateKey()

			if !manager.ValidateKeyFormat(plainKey) {
				t.Error("ValidateKeyFormat() should return true for valid key")
			}
			if manager.ValidateKeyFormat("invalid") {
				t.Error("ValidateKeyFormat() should return false for invalid key")
			}
			if manager.ValidateKeyFormat("plk_tooshort") {
				t.Error("ValidateKeyFormat() should return false for short key")
			}
		},
	)

	t.Run(
		"HashKey", func(t *testing.T) {
			plainKey, expectedHash, _, _ := manager.GenerateKey()

			computedHash := manager.HashKey(plainKey)
			if computedHash != expectedHash {
				t.Error("HashKey() should return same hash for same key")
			}
		},
	)

	t.Run(
		"CreateAPIKey", func(t *testing.T) {
			apiKey, plainKey, err := manager.CreateAPIKey("test-key", "node-1", RoleOperator)
			if err != nil {
				t.Fatalf("CreateAPIKey() error = %v", err)
			}
			if apiKey.Name != "test-key" {
				t.Errorf("apiKey.Name = %v, want %v", apiKey.Name, "test-key")
			}
			if apiKey.NodeID != "node-1" {
				t.Errorf("apiKey.NodeID = %v, want %v", apiKey.NodeID, "node-1")
			}
			if apiKey.Role != RoleOperator {
				t.Errorf("apiKey.Role = %v, want %v", apiKey.Role, RoleOperator)
			}
			if plainKey == "" {
				t.Error("plainKey should not be empty")
			}
		},
	)
}

func TestPasswordHashing(t *testing.T) {
	t.Run(
		"HashPassword", func(t *testing.T) {
			hash, err := HashPassword("testpassword123")
			if err != nil {
				t.Fatalf("HashPassword() error = %v", err)
			}
			if hash == "" {
				t.Error("HashPassword() returned empty hash")
			}
		},
	)

	t.Run(
		"HashPassword_TooShort", func(t *testing.T) {
			_, err := HashPassword("short")
			if err == nil {
				t.Error("HashPassword() should fail for short password")
			}
		},
	)

	t.Run(
		"VerifyPassword", func(t *testing.T) {
			password := "testpassword123"
			hash, _ := HashPassword(password)

			if !VerifyPassword(password, hash) {
				t.Error("VerifyPassword() should return true for correct password")
			}
			if VerifyPassword("wrongpassword", hash) {
				t.Error("VerifyPassword() should return false for wrong password")
			}
		},
	)
}

func TestInMemoryAuthStore(t *testing.T) {
	store := NewInMemoryAuthStore()

	t.Run(
		"User_CRUD", func(t *testing.T) {
			user := User{
				ID:           "user-1",
				Username:     "testuser",
				PasswordHash: "hash",
				Role:         RoleAdmin,
				CreatedAt:    time.Now(),
			}

			if err := store.AddUser(user); err != nil {
				t.Fatalf("AddUser() error = %v", err)
			}

			if err := store.AddUser(user); err != ErrUserAlreadyExists {
				t.Errorf("AddUser() should fail with ErrUserAlreadyExists, got %v", err)
			}

			got, err := store.GetUser("user-1")
			if err != nil {
				t.Fatalf("GetUser() error = %v", err)
			}
			if got.Username != user.Username {
				t.Errorf("GetUser() username = %v, want %v", got.Username, user.Username)
			}

			got, err = store.GetUserByUsername("testuser")
			if err != nil {
				t.Fatalf("GetUserByUsername() error = %v", err)
			}
			if got.ID != user.ID {
				t.Errorf("GetUserByUsername() ID = %v, want %v", got.ID, user.ID)
			}

			newRole := RoleViewer
			if err := store.UpdateUser("user-1", UserUpdate{Role: &newRole}); err != nil {
				t.Fatalf("UpdateUser() error = %v", err)
			}
			got, _ = store.GetUser("user-1")
			if got.Role != RoleViewer {
				t.Errorf("UpdateUser() role = %v, want %v", got.Role, RoleViewer)
			}

			users, err := store.ListUsers()
			if err != nil {
				t.Fatalf("ListUsers() error = %v", err)
			}
			if len(users) != 1 {
				t.Errorf("ListUsers() len = %v, want 1", len(users))
			}

			if err := store.DeleteUser("user-1"); err != nil {
				t.Fatalf("DeleteUser() error = %v", err)
			}

			_, err = store.GetUser("user-1")
			if err != ErrUserNotFound {
				t.Errorf("GetUser() should return ErrUserNotFound, got %v", err)
			}
		},
	)

	t.Run(
		"APIKey_CRUD", func(t *testing.T) {
			apiKey := APIKey{
				ID:        "ak-1",
				KeyHash:   "hash123",
				KeyPrefix: "plk_abc",
				Name:      "test-key",
				Role:      RoleOperator,
				CreatedAt: time.Now(),
			}

			if err := store.AddAPIKey(apiKey); err != nil {
				t.Fatalf("AddAPIKey() error = %v", err)
			}

			got, err := store.GetAPIKey("ak-1")
			if err != nil {
				t.Fatalf("GetAPIKey() error = %v", err)
			}
			if got.Name != apiKey.Name {
				t.Errorf("GetAPIKey() name = %v, want %v", got.Name, apiKey.Name)
			}

			got, err = store.GetAPIKeyByHash("hash123")
			if err != nil {
				t.Fatalf("GetAPIKeyByHash() error = %v", err)
			}
			if got.ID != apiKey.ID {
				t.Errorf("GetAPIKeyByHash() ID = %v, want %v", got.ID, apiKey.ID)
			}

			if err := store.UpdateAPIKeyLastUsed("ak-1"); err != nil {
				t.Fatalf("UpdateAPIKeyLastUsed() error = %v", err)
			}
			got, _ = store.GetAPIKey("ak-1")
			if got.LastUsed == nil {
				t.Error("UpdateAPIKeyLastUsed() should set LastUsed")
			}

			if err := store.RevokeAPIKey("ak-1"); err != nil {
				t.Fatalf("RevokeAPIKey() error = %v", err)
			}
			got, _ = store.GetAPIKey("ak-1")
			if !got.Revoked {
				t.Error("RevokeAPIKey() should set Revoked to true")
			}

			keys, err := store.ListAPIKeys()
			if err != nil {
				t.Fatalf("ListAPIKeys() error = %v", err)
			}
			if len(keys) != 1 {
				t.Errorf("ListAPIKeys() len = %v, want 1", len(keys))
			}

			if err := store.DeleteAPIKey("ak-1"); err != nil {
				t.Fatalf("DeleteAPIKey() error = %v", err)
			}
		},
	)
}

func TestMiddleware(t *testing.T) {
	authStore := NewInMemoryAuthStore()
	config := Config{
		Enabled:      true,
		JWTSecret:    "test-jwt-secret",
		APIKeySecret: "test-apikey-secret",
		TokenExpiry:  time.Hour,
	}

	middleware := NewMiddleware(config, authStore)
	middleware.SetSkipPaths("/public")

	keyManager := middleware.APIKeyManager()
	apiKey, plainKey, _ := keyManager.CreateAPIKey("test-key", "", RoleOperator)
	_ = authStore.AddAPIKey(*apiKey)

	jwtManager := middleware.JWTManager()
	user := &User{ID: "user-1", Username: "testuser", Role: RoleAdmin}
	token, _, _ := jwtManager.GenerateToken(user)

	e := echo.New()
	e.Use(middleware.Authenticate())
	e.GET(
		"/test", func(c echo.Context) error {
			authCtx := GetAuthContext(c)
			if authCtx == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "no auth context"})
			}
			return c.JSON(http.StatusOK, map[string]string{"role": string(authCtx.Role)})
		},
	)
	e.GET(
		"/public", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "public"})
		},
	)

	t.Run(
		"NoAuth", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %v, want %v", rec.Code, http.StatusUnauthorized)
			}
		},
	)

	t.Run(
		"ValidBearerToken", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %v, want %v", rec.Code, http.StatusOK)
			}
		},
	)

	t.Run(
		"ValidAPIKey", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-API-Key", plainKey)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %v, want %v", rec.Code, http.StatusOK)
			}
		},
	)

	t.Run(
		"InvalidToken", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer invalid-token")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %v, want %v", rec.Code, http.StatusUnauthorized)
			}
		},
	)

	t.Run(
		"SkippedPath", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/public", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %v, want %v", rec.Code, http.StatusOK)
			}
		},
	)
}

func TestMiddleware_Disabled(t *testing.T) {
	authStore := NewInMemoryAuthStore()
	config := Config{
		Enabled: false,
	}

	middleware := NewMiddleware(config, authStore)

	e := echo.New()
	e.Use(middleware.Authenticate())
	e.GET(
		"/test", func(c echo.Context) error {
			authCtx := GetAuthContext(c)
			if authCtx == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "no auth context"})
			}
			return c.JSON(http.StatusOK, map[string]string{"role": string(authCtx.Role)})
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %v, want %v", rec.Code, http.StatusOK)
	}
}
