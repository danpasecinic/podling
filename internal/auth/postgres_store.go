package auth

import (
	"database/sql"
	"time"
)

type PostgresAuthStore struct {
	db *sql.DB
}

func NewPostgresAuthStore(db *sql.DB) *PostgresAuthStore {
	return &PostgresAuthStore{db: db}
}

func (s *PostgresAuthStore) AddUser(user User) error {
	query := `
		INSERT INTO users (id, username, password_hash, role, created_at, disabled)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.Exec(query, user.ID, user.Username, user.PasswordHash, user.Role, user.CreatedAt, user.Disabled)
	if err != nil {
		return ErrUserAlreadyExists
	}
	return nil
}

func (s *PostgresAuthStore) GetUser(userID string) (User, error) {
	query := `SELECT id, username, password_hash, role, created_at, last_login, disabled FROM users WHERE id = $1`
	var user User
	var lastLogin sql.NullTime
	err := s.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin, &user.Disabled,
	)
	if err == sql.ErrNoRows {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	return user, nil
}

func (s *PostgresAuthStore) GetUserByUsername(username string) (User, error) {
	query := `SELECT id, username, password_hash, role, created_at, last_login, disabled FROM users WHERE username = $1`
	var user User
	var lastLogin sql.NullTime
	err := s.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin, &user.Disabled,
	)
	if err == sql.ErrNoRows {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	return user, nil
}

func (s *PostgresAuthStore) UpdateUser(userID string, updates UserUpdate) error {
	if updates.PasswordHash != nil {
		if _, err := s.db.Exec(
			`UPDATE users SET password_hash = $1 WHERE id = $2`, *updates.PasswordHash, userID,
		); err != nil {
			return err
		}
	}
	if updates.Role != nil {
		if _, err := s.db.Exec(`UPDATE users SET role = $1 WHERE id = $2`, *updates.Role, userID); err != nil {
			return err
		}
	}
	if updates.LastLogin != nil {
		if _, err := s.db.Exec(
			`UPDATE users SET last_login = $1 WHERE id = $2`, *updates.LastLogin, userID,
		); err != nil {
			return err
		}
	}
	if updates.Disabled != nil {
		if _, err := s.db.Exec(`UPDATE users SET disabled = $1 WHERE id = $2`, *updates.Disabled, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresAuthStore) ListUsers() ([]User, error) {
	query := `SELECT id, username, password_hash, role, created_at, last_login, disabled FROM users`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var users []User
	for rows.Next() {
		var user User
		var lastLogin sql.NullTime
		if err := rows.Scan(
			&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &lastLogin, &user.Disabled,
		); err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *PostgresAuthStore) DeleteUser(userID string) error {
	result, err := s.db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *PostgresAuthStore) AddAPIKey(apiKey APIKey) error {
	query := `
		INSERT INTO api_keys (id, key_hash, key_prefix, name, node_id, role, created_at, expires_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.db.Exec(
		query, apiKey.ID, apiKey.KeyHash, apiKey.KeyPrefix, apiKey.Name,
		sql.NullString{String: apiKey.NodeID, Valid: apiKey.NodeID != ""},
		apiKey.Role, apiKey.CreatedAt, apiKey.ExpiresAt, apiKey.Revoked,
	)
	return err
}

func (s *PostgresAuthStore) GetAPIKey(keyID string) (APIKey, error) {
	query := `SELECT id, key_hash, key_prefix, name, node_id, role, created_at, expires_at, last_used, revoked FROM api_keys WHERE id = $1`
	var apiKey APIKey
	var nodeID sql.NullString
	var expiresAt, lastUsed sql.NullTime
	err := s.db.QueryRow(query, keyID).Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.KeyPrefix, &apiKey.Name, &nodeID,
		&apiKey.Role, &apiKey.CreatedAt, &expiresAt, &lastUsed, &apiKey.Revoked,
	)
	if err == sql.ErrNoRows {
		return APIKey{}, ErrAPIKeyNotFound
	}
	if err != nil {
		return APIKey{}, err
	}
	if nodeID.Valid {
		apiKey.NodeID = nodeID.String
	}
	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}
	if lastUsed.Valid {
		apiKey.LastUsed = &lastUsed.Time
	}
	return apiKey, nil
}

func (s *PostgresAuthStore) GetAPIKeyByHash(keyHash string) (APIKey, error) {
	query := `SELECT id, key_hash, key_prefix, name, node_id, role, created_at, expires_at, last_used, revoked FROM api_keys WHERE key_hash = $1`
	var apiKey APIKey
	var nodeID sql.NullString
	var expiresAt, lastUsed sql.NullTime
	err := s.db.QueryRow(query, keyHash).Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.KeyPrefix, &apiKey.Name, &nodeID,
		&apiKey.Role, &apiKey.CreatedAt, &expiresAt, &lastUsed, &apiKey.Revoked,
	)
	if err == sql.ErrNoRows {
		return APIKey{}, ErrAPIKeyNotFound
	}
	if err != nil {
		return APIKey{}, err
	}
	if nodeID.Valid {
		apiKey.NodeID = nodeID.String
	}
	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}
	if lastUsed.Valid {
		apiKey.LastUsed = &lastUsed.Time
	}
	return apiKey, nil
}

func (s *PostgresAuthStore) UpdateAPIKeyLastUsed(keyID string) error {
	_, err := s.db.Exec(`UPDATE api_keys SET last_used = $1 WHERE id = $2`, time.Now(), keyID)
	return err
}

func (s *PostgresAuthStore) RevokeAPIKey(keyID string) error {
	result, err := s.db.Exec(`UPDATE api_keys SET revoked = true WHERE id = $1`, keyID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

func (s *PostgresAuthStore) ListAPIKeys() ([]APIKey, error) {
	query := `SELECT id, key_hash, key_prefix, name, node_id, role, created_at, expires_at, last_used, revoked FROM api_keys`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var keys []APIKey
	for rows.Next() {
		var apiKey APIKey
		var nodeID sql.NullString
		var expiresAt, lastUsed sql.NullTime
		if err := rows.Scan(
			&apiKey.ID, &apiKey.KeyHash, &apiKey.KeyPrefix, &apiKey.Name, &nodeID,
			&apiKey.Role, &apiKey.CreatedAt, &expiresAt, &lastUsed, &apiKey.Revoked,
		); err != nil {
			return nil, err
		}
		if nodeID.Valid {
			apiKey.NodeID = nodeID.String
		}
		if expiresAt.Valid {
			apiKey.ExpiresAt = &expiresAt.Time
		}
		if lastUsed.Valid {
			apiKey.LastUsed = &lastUsed.Time
		}
		keys = append(keys, apiKey)
	}
	return keys, nil
}

func (s *PostgresAuthStore) ListAPIKeysByNodeID(nodeID string) ([]APIKey, error) {
	query := `SELECT id, key_hash, key_prefix, name, node_id, role, created_at, expires_at, last_used, revoked FROM api_keys WHERE node_id = $1`
	rows, err := s.db.Query(query, nodeID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var keys []APIKey
	for rows.Next() {
		var apiKey APIKey
		var nID sql.NullString
		var expiresAt, lastUsed sql.NullTime
		if err := rows.Scan(
			&apiKey.ID, &apiKey.KeyHash, &apiKey.KeyPrefix, &apiKey.Name, &nID,
			&apiKey.Role, &apiKey.CreatedAt, &expiresAt, &lastUsed, &apiKey.Revoked,
		); err != nil {
			return nil, err
		}
		if nID.Valid {
			apiKey.NodeID = nID.String
		}
		if expiresAt.Valid {
			apiKey.ExpiresAt = &expiresAt.Time
		}
		if lastUsed.Valid {
			apiKey.LastUsed = &lastUsed.Time
		}
		keys = append(keys, apiKey)
	}
	return keys, nil
}

func (s *PostgresAuthStore) DeleteAPIKey(keyID string) error {
	result, err := s.db.Exec(`DELETE FROM api_keys WHERE id = $1`, keyID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}
