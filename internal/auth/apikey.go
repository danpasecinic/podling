package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	apiKeyPrefix = "plk_"
	keyLength    = 32
)

type APIKeyManager struct {
	secret []byte
}

func NewAPIKeyManager(secret string) *APIKeyManager {
	return &APIKeyManager{
		secret: []byte(secret),
	}
}

func (m *APIKeyManager) GenerateKey() (plainKey string, keyHash string, keyPrefix string, err error) {
	bytes := make([]byte, keyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	rawKey := hex.EncodeToString(bytes)
	plainKey = apiKeyPrefix + rawKey
	keyPrefix = plainKey[:12]
	keyHash = m.HashKey(plainKey)

	return plainKey, keyHash, keyPrefix, nil
}

func (m *APIKeyManager) HashKey(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	h.Write(m.secret)
	return hex.EncodeToString(h.Sum(nil))
}

func (m *APIKeyManager) ValidateKeyFormat(key string) bool {
	if !strings.HasPrefix(key, apiKeyPrefix) {
		return false
	}
	rawKey := strings.TrimPrefix(key, apiKeyPrefix)
	if len(rawKey) != keyLength*2 {
		return false
	}
	_, err := hex.DecodeString(rawKey)
	return err == nil
}

func (m *APIKeyManager) CreateAPIKey(name string, nodeID string, role Role) (*APIKey, string, error) {
	plainKey, keyHash, keyPrefix, err := m.GenerateKey()
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	id := fmt.Sprintf("ak-%s-%s", now.Format("20060102150405"), randomString(8))

	apiKey := &APIKey{
		ID:        id,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Name:      name,
		NodeID:    nodeID,
		Role:      role,
		CreatedAt: now,
		Revoked:   false,
	}

	return apiKey, plainKey, nil
}

func randomString(n int) string {
	bytes := make([]byte, n)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)[:n]
}
