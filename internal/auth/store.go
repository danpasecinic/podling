package auth

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrAPIKeyNotFound    = errors.New("api key not found")
	ErrAPIKeyRevoked     = errors.New("api key has been revoked")
	ErrAPIKeyExpired     = errors.New("api key has expired")
	ErrInvalidAPIKey     = errors.New("invalid api key")
)

type AuthStore interface {
	AddUser(user User) error
	GetUser(userID string) (User, error)
	GetUserByUsername(username string) (User, error)
	UpdateUser(userID string, updates UserUpdate) error
	ListUsers() ([]User, error)
	DeleteUser(userID string) error

	AddAPIKey(apiKey APIKey) error
	GetAPIKey(keyID string) (APIKey, error)
	GetAPIKeyByHash(keyHash string) (APIKey, error)
	UpdateAPIKeyLastUsed(keyID string) error
	RevokeAPIKey(keyID string) error
	ListAPIKeys() ([]APIKey, error)
	ListAPIKeysByNodeID(nodeID string) ([]APIKey, error)
	DeleteAPIKey(keyID string) error
}

type UserUpdate struct {
	PasswordHash *string
	Role         *Role
	LastLogin    *time.Time
	Disabled     *bool
}
