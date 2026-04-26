// Package store - Admin account storage and session management.
// Admin accounts are in users.db (admins table), NOT in config file.
// See AI.md PART 17 for admin panel specifications.
package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

// AdminStore manages admin accounts and sessions in users.db
type AdminStore struct {
	db *sql.DB
}

// NewAdminStore creates a new admin store with its own database
func NewAdminStore(dbPath string) (*AdminStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening users database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &AdminStore{db: db}

	if err := store.ensureSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensuring admin schema: %w", err)
	}

	return store, nil
}

// Close closes the database
func (s *AdminStore) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection
func (s *AdminStore) DB() *sql.DB {
	return s.db
}

func (s *AdminStore) ensureSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			email TEXT NOT NULL DEFAULT '',
			totp_secret TEXT NOT NULL DEFAULT '',
			totp_enabled INTEGER NOT NULL DEFAULT 0,
			recovery_keys TEXT NOT NULL DEFAULT '',
			last_login_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			admin_id INTEGER NOT NULL,
			ip_address TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS setup_tokens (
			token_hash TEXT PRIMARY KEY,
			used INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS api_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id INTEGER NOT NULL,
			name TEXT NOT NULL DEFAULT 'default',
			token_hash TEXT NOT NULL UNIQUE,
			last_used_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE
		)`,

		`CREATE INDEX IF NOT EXISTS idx_sessions_admin ON sessions(admin_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_api_tokens_admin ON api_tokens(admin_id)`,
	}

	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("creating table: %w", err)
		}
	}

	return nil
}

// Admin represents an admin account
type Admin struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	TOTPEnabled  bool       `json:"totp_enabled"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// HasAdmins returns true if at least one admin account exists
func (s *AdminStore) HasAdmins() bool {
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count)
	return count > 0
}

// CreateAdmin creates a new admin account with Argon2id hashed password
func (s *AdminStore) CreateAdmin(username, password string) (int64, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return 0, fmt.Errorf("hashing password: %w", err)
	}

	result, err := s.db.Exec(
		`INSERT INTO admins (username, password_hash) VALUES (?, ?)`,
		username, hash,
	)
	if err != nil {
		return 0, fmt.Errorf("creating admin: %w", err)
	}

	return result.LastInsertId()
}

// VerifyAdminLogin checks username/password and returns the admin if valid
func (s *AdminStore) VerifyAdminLogin(username, password string) (*Admin, error) {
	var admin Admin
	var passwordHash string

	err := s.db.QueryRow(
		`SELECT id, username, email, password_hash, totp_enabled, last_login_at, created_at
		 FROM admins WHERE username = ?`, username,
	).Scan(&admin.ID, &admin.Username, &admin.Email, &passwordHash,
		&admin.TOTPEnabled, &admin.LastLoginAt, &admin.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying admin: %w", err)
	}

	if !verifyPassword(password, passwordHash) {
		return nil, nil
	}

	// Update last login
	s.db.Exec(`UPDATE admins SET last_login_at = ? WHERE id = ?`, time.Now(), admin.ID)

	return &admin, nil
}

// CreateSession creates a new admin session
func (s *AdminStore) CreateSession(adminID int64, ipAddress, userAgent string, duration time.Duration) (string, error) {
	sessionID, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("generating session ID: %w", err)
	}

	expiresAt := time.Now().Add(duration)

	_, err = s.db.Exec(
		`INSERT INTO sessions (id, admin_id, ip_address, user_agent, expires_at) VALUES (?, ?, ?, ?, ?)`,
		sessionID, adminID, ipAddress, userAgent, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}

	return sessionID, nil
}

// GetSession validates a session and returns the admin
func (s *AdminStore) GetSession(sessionID string) (*Admin, error) {
	var admin Admin

	err := s.db.QueryRow(
		`SELECT a.id, a.username, a.email, a.totp_enabled, a.last_login_at, a.created_at
		 FROM sessions s JOIN admins a ON s.admin_id = a.id
		 WHERE s.id = ? AND s.expires_at > ?`, sessionID, time.Now(),
	).Scan(&admin.ID, &admin.Username, &admin.Email, &admin.TOTPEnabled,
		&admin.LastLoginAt, &admin.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}

	return &admin, nil
}

// DeleteSession removes a session (logout)
func (s *AdminStore) DeleteSession(sessionID string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

// GenerateSetupToken creates a one-time setup token
func (s *AdminStore) GenerateSetupToken() (string, error) {
	token, err := generateSecureToken(16)
	if err != nil {
		return "", fmt.Errorf("generating setup token: %w", err)
	}

	tokenHash := hashToken(token)
	_, err = s.db.Exec(
		`INSERT INTO setup_tokens (token_hash) VALUES (?)`, tokenHash,
	)
	if err != nil {
		return "", fmt.Errorf("storing setup token: %w", err)
	}

	return token, nil
}

// ValidateSetupToken checks if a setup token is valid and unused
func (s *AdminStore) ValidateSetupToken(token string) (bool, error) {
	tokenHash := hashToken(token)

	var used int
	err := s.db.QueryRow(
		`SELECT used FROM setup_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&used)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("validating setup token: %w", err)
	}

	return used == 0, nil
}

// UseSetupToken marks a setup token as used
func (s *AdminStore) UseSetupToken(token string) error {
	tokenHash := hashToken(token)
	_, err := s.db.Exec(`UPDATE setup_tokens SET used = 1 WHERE token_hash = ?`, tokenHash)
	return err
}

// GenerateAPIToken creates an API token for an admin
func (s *AdminStore) GenerateAPIToken(adminID int64, name string) (string, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("generating API token: %w", err)
	}

	// Prefix with adm_ per spec
	prefixedToken := "adm_" + token
	tokenHashValue := hashToken(prefixedToken)

	_, err = s.db.Exec(
		`INSERT INTO api_tokens (admin_id, name, token_hash) VALUES (?, ?, ?)`,
		adminID, name, tokenHashValue,
	)
	if err != nil {
		return "", fmt.Errorf("storing API token: %w", err)
	}

	return prefixedToken, nil
}

// ValidateAPIToken checks if an API token is valid and returns the admin
func (s *AdminStore) ValidateAPIToken(token string) (*Admin, error) {
	tokenHashValue := hashToken(token)

	var admin Admin
	err := s.db.QueryRow(
		`SELECT a.id, a.username, a.email, a.totp_enabled, a.created_at
		 FROM api_tokens t JOIN admins a ON t.admin_id = a.id
		 WHERE t.token_hash = ?`, tokenHashValue,
	).Scan(&admin.ID, &admin.Username, &admin.Email, &admin.TOTPEnabled, &admin.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("validating API token: %w", err)
	}

	// Update last used
	s.db.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE token_hash = ?`, time.Now(), tokenHashValue)

	return &admin, nil
}

// Argon2id parameters (OWASP 2023)
const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// hashPassword hashes a password using Argon2id
func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	), nil
}

// verifyPassword checks a password against an Argon2id hash
func verifyPassword(password, encodedHash string) bool {
	// Parse the encoded hash
	var memory uint32
	var iterations uint32
	var parallelism uint8
	var saltHex, hashHex string

	_, err := fmt.Sscanf(encodedHash, "$argon2id$v=19$m=%d,t=%d,p=%d$%s",
		&memory, &iterations, &parallelism, &saltHex)
	if err != nil {
		return false
	}

	// Split salt and hash from the remaining string
	parts := splitLast(saltHex, "$")
	if len(parts) != 2 {
		return false
	}
	saltHex = parts[0]
	hashHex = parts[1]

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return false
	}

	expectedHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return false
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedHash)))

	// Constant-time comparison
	if len(computedHash) != len(expectedHash) {
		return false
	}
	var diff byte
	for i := range computedHash {
		diff |= computedHash[i] ^ expectedHash[i]
	}
	return diff == 0
}

// hashToken hashes a token using SHA-256 (fast lookup for high-entropy tokens)
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// generateSecureToken generates a cryptographically secure random hex token
func generateSecureToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func splitLast(s, sep string) []string {
	idx := -1
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}
