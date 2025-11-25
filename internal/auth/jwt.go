package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret        []byte
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
}

func NewJWTManager(secret string, tokenExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secret:        []byte(secret),
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (m *JWTManager) GenerateToken(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.tokenExpiry)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "podling",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, expiresAt, nil
}

func (m *JWTManager) GenerateRefreshToken(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.refreshExpiry)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "podling-refresh",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signed, expiresAt, nil
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (m *JWTManager) RefreshToken(refreshTokenString string) (string, time.Time, error) {
	claims, err := m.ValidateToken(refreshTokenString)
	if err != nil {
		return "", time.Time{}, err
	}

	if claims.Issuer != "podling-refresh" {
		return "", time.Time{}, fmt.Errorf("invalid refresh token")
	}

	user := &User{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	}

	return m.GenerateToken(user)
}
