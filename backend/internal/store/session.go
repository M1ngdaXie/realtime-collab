package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/google/uuid"
)

// CreateSession creates a new session
func (s *PostgresStore) CreateSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, token string, expiresAt time.Time, userAgent, ipAddress *string) (*models.Session, error) {
    // Hash the token before storing
    hash := sha256.Sum256([]byte(token))
    tokenHash := hex.EncodeToString(hash[:])

    // 【修改点 1】: 在 SQL 里显式加上 id 字段
    // 注意：$1 变成了 id，后面的参数序号全部要顺延 (+1)
    query := `         
        INSERT INTO sessions (id, user_id, token_hash, expires_at, user_agent, ip_address)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, user_id, token_hash, expires_at, created_at, user_agent, ip_address
    `

    var session models.Session
    // 【修改点 2】: 在参数列表的最前面加上 sessionID
    // 现在的对应关系：
    // $1 -> sessionID
    // $2 -> userID
    // $3 -> tokenHash
    // ...
    err := s.db.QueryRowContext(ctx, query, 
        sessionID, // <--- 这里！把它传进去！
        userID, 
        tokenHash, 
        expiresAt, 
        userAgent, 
        ipAddress,
    ).Scan(
        &session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt,
        &session.CreatedAt, &session.UserAgent, &session.IPAddress,
    )
    if err != nil {
        return nil, err
    }

    return &session, nil
}

// GetSessionByToken retrieves session by JWT token
func (s *PostgresStore) GetSessionByToken(ctx context.Context, token string) (*models.Session, error) {
    hash := sha256.Sum256([]byte(token))
    tokenHash := hex.EncodeToString(hash[:])

    query := `
        SELECT id, user_id, token_hash, expires_at, created_at, user_agent, ip_address
        FROM sessions
        WHERE token_hash = $1 AND expires_at > NOW()
    `

    var session models.Session
    err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
        &session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt,
        &session.CreatedAt, &session.UserAgent, &session.IPAddress,
    )
    if err != nil {
        return nil, err
    }

    return &session, nil
}

// DeleteSession deletes a session (logout)
func (s *PostgresStore) DeleteSession(ctx context.Context, token string) error {
    hash := sha256.Sum256([]byte(token))
    tokenHash := hex.EncodeToString(hash[:])

    _, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = $1", tokenHash)
    return err
}

// CleanupExpiredSessions removes expired sessions
func (s *PostgresStore) CleanupExpiredSessions(ctx context.Context) error {
    _, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < NOW()")
    return err
}
