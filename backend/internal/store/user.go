package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/google/uuid"
)

// UpsertUser creates or updates user from OAuth profile
func (s *PostgresStore) UpsertUser(ctx context.Context, provider, providerUserID, email, name string, avatarURL *string) (*models.User, error) {
    query := `
        INSERT INTO users (provider, provider_user_id, email, name, avatar_url, last_login_at)
        VALUES ($1, $2, $3, $4, $5, NOW())
        ON CONFLICT (provider, provider_user_id)
        DO UPDATE SET
            email = EXCLUDED.email,
            name = EXCLUDED.name,
            avatar_url = EXCLUDED.avatar_url,
            last_login_at = NOW(),
            updated_at = NOW()
        RETURNING id, email, name, avatar_url, provider, provider_user_id, created_at, updated_at, last_login_at
    `

    var user models.User
    err := s.db.QueryRowContext(ctx, query, provider, providerUserID, email, name, avatarURL).Scan(
        &user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.Provider,
        &user.ProviderUserID, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
    )
    if err != nil {
        return nil, err
    }
    fmt.Printf("✅ User Upserted: ID=%s, Email=%s\n", user.ID.String(), user.Email)

    return &user, nil
}

// GetUserByID retrieves user by ID
func (s *PostgresStore) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
    query := `
        SELECT id, email, name, avatar_url, provider, provider_user_id, created_at, updated_at, last_login_at
        FROM users WHERE id = $1
    `

    var user models.User
    err := s.db.QueryRowContext(ctx, query, userID).Scan(
        &user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.Provider,
        &user.ProviderUserID, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    return &user, nil
}

// GetUserByEmail retrieves user by email
func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    query := `
        SELECT id, email, name, avatar_url, provider, provider_user_id, created_at, updated_at, last_login_at
        FROM users WHERE email = $1
    `

    var user models.User
    err := s.db.QueryRowContext(ctx, query, email).Scan(
        &user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.Provider,
        &user.ProviderUserID, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    return &user, nil
}
