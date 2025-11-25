package auth

import (
	"errors"
	"sync"
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

type InMemoryAuthStore struct {
	mu      sync.RWMutex
	users   map[string]User
	apiKeys map[string]APIKey
}

func NewInMemoryAuthStore() *InMemoryAuthStore {
	return &InMemoryAuthStore{
		users:   make(map[string]User),
		apiKeys: make(map[string]APIKey),
	}
}

func (s *InMemoryAuthStore) AddUser(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; exists {
		return ErrUserAlreadyExists
	}

	for _, u := range s.users {
		if u.Username == user.Username {
			return ErrUserAlreadyExists
		}
	}

	s.users[user.ID] = user
	return nil
}

func (s *InMemoryAuthStore) GetUser(userID string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return User{}, ErrUserNotFound
	}

	return user, nil
}

func (s *InMemoryAuthStore) GetUserByUsername(username string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Username == username {
			return user, nil
		}
	}

	return User{}, ErrUserNotFound
}

func (s *InMemoryAuthStore) UpdateUser(userID string, updates UserUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return ErrUserNotFound
	}

	if updates.PasswordHash != nil {
		user.PasswordHash = *updates.PasswordHash
	}
	if updates.Role != nil {
		user.Role = *updates.Role
	}
	if updates.LastLogin != nil {
		user.LastLogin = updates.LastLogin
	}
	if updates.Disabled != nil {
		user.Disabled = *updates.Disabled
	}

	s.users[userID] = user
	return nil
}

func (s *InMemoryAuthStore) ListUsers() ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	return users, nil
}

func (s *InMemoryAuthStore) DeleteUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[userID]; !exists {
		return ErrUserNotFound
	}

	delete(s.users, userID)
	return nil
}

func (s *InMemoryAuthStore) AddAPIKey(apiKey APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apiKeys[apiKey.ID]; exists {
		return errors.New("api key already exists")
	}

	s.apiKeys[apiKey.ID] = apiKey
	return nil
}

func (s *InMemoryAuthStore) GetAPIKey(keyID string) (APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apiKey, exists := s.apiKeys[keyID]
	if !exists {
		return APIKey{}, ErrAPIKeyNotFound
	}

	return apiKey, nil
}

func (s *InMemoryAuthStore) GetAPIKeyByHash(keyHash string) (APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, apiKey := range s.apiKeys {
		if apiKey.KeyHash == keyHash {
			return apiKey, nil
		}
	}

	return APIKey{}, ErrAPIKeyNotFound
}

func (s *InMemoryAuthStore) UpdateAPIKeyLastUsed(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	apiKey, exists := s.apiKeys[keyID]
	if !exists {
		return ErrAPIKeyNotFound
	}

	now := time.Now()
	apiKey.LastUsed = &now
	s.apiKeys[keyID] = apiKey
	return nil
}

func (s *InMemoryAuthStore) RevokeAPIKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	apiKey, exists := s.apiKeys[keyID]
	if !exists {
		return ErrAPIKeyNotFound
	}

	apiKey.Revoked = true
	s.apiKeys[keyID] = apiKey
	return nil
}

func (s *InMemoryAuthStore) ListAPIKeys() ([]APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]APIKey, 0, len(s.apiKeys))
	for _, key := range s.apiKeys {
		keys = append(keys, key)
	}

	return keys, nil
}

func (s *InMemoryAuthStore) ListAPIKeysByNodeID(nodeID string) ([]APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]APIKey, 0)
	for _, key := range s.apiKeys {
		if key.NodeID == nodeID {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (s *InMemoryAuthStore) DeleteAPIKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apiKeys[keyID]; !exists {
		return ErrAPIKeyNotFound
	}

	delete(s.apiKeys, keyID)
	return nil
}
