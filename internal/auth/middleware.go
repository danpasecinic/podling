package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	AuthContextKey = "auth"
	AuthTypeJWT    = "jwt"
	AuthTypeAPIKey = "apikey"
)

type Middleware struct {
	config     Config
	jwtManager *JWTManager
	keyManager *APIKeyManager
	authStore  AuthStore
	skipPaths  map[string]bool
}

func NewMiddleware(config Config, authStore AuthStore) *Middleware {
	m := &Middleware{
		config:    config,
		authStore: authStore,
		skipPaths: make(map[string]bool),
	}

	if config.JWTSecret != "" {
		m.jwtManager = NewJWTManager(config.JWTSecret, config.TokenExpiry, config.RefreshExpiry)
	}
	if config.APIKeySecret != "" {
		m.keyManager = NewAPIKeyManager(config.APIKeySecret)
	}

	return m
}

func (m *Middleware) SetSkipPaths(paths ...string) {
	for _, p := range paths {
		m.skipPaths[p] = true
	}
}

func (m *Middleware) JWTManager() *JWTManager {
	return m.jwtManager
}

func (m *Middleware) APIKeyManager() *APIKeyManager {
	return m.keyManager
}

func (m *Middleware) Authenticate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !m.config.Enabled {
				c.Set(
					AuthContextKey, &AuthContext{
						Role:     RoleAdmin,
						AuthType: "disabled",
					},
				)
				return next(c)
			}

			path := c.Path()
			if m.skipPaths[path] {
				return next(c)
			}

			authHeader := c.Request().Header.Get("Authorization")
			apiKeyHeader := c.Request().Header.Get("X-API-Key")

			if authHeader == "" && apiKeyHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authentication")
			}

			var authCtx *AuthContext
			var err error

			if apiKeyHeader != "" {
				authCtx, err = m.validateAPIKey(apiKeyHeader)
			} else if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				authCtx, err = m.validateJWT(token)
			} else if strings.HasPrefix(authHeader, "ApiKey ") {
				key := strings.TrimPrefix(authHeader, "ApiKey ")
				authCtx, err = m.validateAPIKey(key)
			} else {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid authorization header format")
			}

			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			}

			c.Set(AuthContextKey, authCtx)
			return next(c)
		}
	}
}

func (m *Middleware) validateJWT(tokenString string) (*AuthContext, error) {
	if m.jwtManager == nil {
		return nil, ErrInvalidAPIKey
	}

	claims, err := m.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &AuthContext{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		AuthType: AuthTypeJWT,
	}, nil
}

func (m *Middleware) validateAPIKey(key string) (*AuthContext, error) {
	if m.keyManager == nil {
		return nil, ErrInvalidAPIKey
	}

	if !m.keyManager.ValidateKeyFormat(key) {
		return nil, ErrInvalidAPIKey
	}

	keyHash := m.keyManager.HashKey(key)
	apiKey, err := m.authStore.GetAPIKeyByHash(keyHash)
	if err != nil {
		return nil, ErrInvalidAPIKey
	}

	if apiKey.Revoked {
		return nil, ErrAPIKeyRevoked
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, ErrAPIKeyExpired
	}

	go func() {
		_ = m.authStore.UpdateAPIKeyLastUsed(apiKey.ID)
	}()

	return &AuthContext{
		UserID:   apiKey.ID,
		Username: apiKey.Name,
		Role:     apiKey.Role,
		NodeID:   apiKey.NodeID,
		AuthType: AuthTypeAPIKey,
	}, nil
}

func (m *Middleware) RequireRole(roles ...Role) echo.MiddlewareFunc {
	roleSet := make(map[Role]bool)
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authCtx := GetAuthContext(c)
			if authCtx == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
			}

			if !roleSet[authCtx.Role] {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}

			return next(c)
		}
	}
}

func (m *Middleware) RequireWrite() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authCtx := GetAuthContext(c)
			if authCtx == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
			}

			if !authCtx.Role.CanWrite() {
				return echo.NewHTTPError(http.StatusForbidden, "write access required")
			}

			return next(c)
		}
	}
}

func (m *Middleware) RequireDelete() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authCtx := GetAuthContext(c)
			if authCtx == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
			}

			if !authCtx.Role.CanDelete() {
				return echo.NewHTTPError(http.StatusForbidden, "delete access required")
			}

			return next(c)
		}
	}
}

func (m *Middleware) RequireNodeManagement() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authCtx := GetAuthContext(c)
			if authCtx == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
			}

			if !authCtx.Role.CanManageNodes() {
				return echo.NewHTTPError(http.StatusForbidden, "node management access required")
			}

			return next(c)
		}
	}
}

func GetAuthContext(c echo.Context) *AuthContext {
	if authCtx, ok := c.Get(AuthContextKey).(*AuthContext); ok {
		return authCtx
	}
	return nil
}
