package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/APICerberus/APICerebrus/internal/pkg/uuid"
)

type Session struct {
	ID        string
	UserID    string
	TokenHash string
	UserAgent string
	ClientIP  string
	ExpiresAt time.Time
	LastSeen  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SessionRepo struct {
	db  *sql.DB
	now func() time.Time
}

func (s *Store) Sessions() *SessionRepo {
	if s == nil || s.db == nil {
		return nil
	}
	return &SessionRepo{
		db:  s.db,
		now: time.Now,
	}
}

func GenerateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func (r *SessionRepo) Create(session *Session) error {
	if r == nil || r.db == nil {
		return errors.New("session repo is not initialized")
	}
	if session == nil {
		return errors.New("session is nil")
	}
	session.UserID = strings.TrimSpace(session.UserID)
	session.TokenHash = strings.TrimSpace(strings.ToLower(session.TokenHash))
	if session.UserID == "" {
		return errors.New("session user_id is required")
	}
	if session.TokenHash == "" {
		return errors.New("session token_hash is required")
	}
	if session.ExpiresAt.IsZero() {
		return errors.New("session expires_at is required")
	}
	if strings.TrimSpace(session.ID) == "" {
		id, err := uuid.NewString()
		if err != nil {
			return err
		}
		session.ID = id
	}

	now := r.now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.LastSeen.IsZero() {
		session.LastSeen = now
	}
	session.UpdatedAt = now

	_, err := r.db.Exec(`
		INSERT INTO sessions(
			id, user_id, token_hash, user_agent, client_ip, expires_at, last_seen_at, created_at, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		session.ID,
		session.UserID,
		session.TokenHash,
		strings.TrimSpace(session.UserAgent),
		strings.TrimSpace(session.ClientIP),
		session.ExpiresAt.UTC().Format(time.RFC3339Nano),
		session.LastSeen.UTC().Format(time.RFC3339Nano),
		session.CreatedAt.UTC().Format(time.RFC3339Nano),
		session.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *SessionRepo) FindByTokenHash(tokenHash string) (*Session, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("session repo is not initialized")
	}
	tokenHash = strings.TrimSpace(strings.ToLower(tokenHash))
	if tokenHash == "" {
		return nil, errors.New("token hash is required")
	}

	row := r.db.QueryRow(`
		SELECT id, user_id, token_hash, user_agent, client_ip, expires_at, last_seen_at, created_at, updated_at
		  FROM sessions
		 WHERE token_hash = ?
	`, tokenHash)
	session, err := scanSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return session, nil
}

func (r *SessionRepo) DeleteByID(id string) error {
	if r == nil || r.db == nil {
		return errors.New("session repo is not initialized")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("session id is required")
	}

	_, err := r.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session by id: %w", err)
	}
	return nil
}

func (r *SessionRepo) DeleteByTokenHash(tokenHash string) error {
	if r == nil || r.db == nil {
		return errors.New("session repo is not initialized")
	}
	tokenHash = strings.TrimSpace(strings.ToLower(tokenHash))
	if tokenHash == "" {
		return errors.New("token hash is required")
	}

	_, err := r.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session by token hash: %w", err)
	}
	return nil
}

func (r *SessionRepo) Touch(id string) error {
	if r == nil || r.db == nil {
		return errors.New("session repo is not initialized")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("session id is required")
	}

	now := r.now().UTC().Format(time.RFC3339Nano)
	result, err := r.db.Exec(`UPDATE sessions SET last_seen_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SessionRepo) CleanupExpired(now time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("session repo is not initialized")
	}
	if now.IsZero() {
		now = r.now()
	}
	result, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("cleanup expired sessions: %w", err)
	}
	deleted, _ := result.RowsAffected()
	return deleted, nil
}

func scanSession(row *sql.Row) (*Session, error) {
	var (
		session                                             Session
		expiresAtRaw, lastSeenRaw, createdAtRaw, updatedRaw string
	)
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.ClientIP,
		&expiresAtRaw,
		&lastSeenRaw,
		&createdAtRaw,
		&updatedRaw,
	); err != nil {
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtRaw)
	if err != nil {
		return nil, fmt.Errorf("decode session expires_at: %w", err)
	}
	lastSeen, err := time.Parse(time.RFC3339Nano, lastSeenRaw)
	if err != nil {
		return nil, fmt.Errorf("decode session last_seen_at: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return nil, fmt.Errorf("decode session created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		return nil, fmt.Errorf("decode session updated_at: %w", err)
	}

	session.ExpiresAt = expiresAt
	session.LastSeen = lastSeen
	session.CreatedAt = createdAt
	session.UpdatedAt = updatedAt
	return &session, nil
}
