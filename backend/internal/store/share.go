package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/google/uuid"
)

// CreateDocumentShare creates a new share
func (s *PostgresStore) CreateDocumentShare(ctx context.Context, documentID, userID uuid.UUID, permission string, createdBy *uuid.UUID) (*models.DocumentShare, error) {
    query := `
        INSERT INTO document_shares (document_id, user_id, permission, created_by)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (document_id, user_id) DO UPDATE SET permission = EXCLUDED.permission
        RETURNING id, document_id, user_id, permission, created_at, created_by
    `

    var share models.DocumentShare
    err := s.db.QueryRowContext(ctx, query, documentID, userID, permission, createdBy).Scan(
        &share.ID, &share.DocumentID, &share.UserID, &share.Permission,
        &share.CreatedAt, &share.CreatedBy,
    )
    if err != nil {
        return nil, err
    }

    return &share, nil
}

// ListDocumentShares lists all shares for a document
func (s *PostgresStore) ListDocumentShares(ctx context.Context, documentID uuid.UUID) ([]models.DocumentShareWithUser, error) {
    query := `
        SELECT
            ds.id, ds.document_id, ds.user_id, ds.permission, ds.created_at, ds.created_by,
            u.id, u.email, u.name, u.avatar_url, u.provider, u.provider_user_id, u.created_at, u.updated_at, u.last_login_at
        FROM document_shares ds
        JOIN users u ON ds.user_id = u.id
        WHERE ds.document_id = $1
        ORDER BY ds.created_at DESC
    `

    rows, err := s.db.QueryContext(ctx, query, documentID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var shares []models.DocumentShareWithUser
    for rows.Next() {
        var share models.DocumentShareWithUser
        err := rows.Scan(
            &share.ID, &share.DocumentID, &share.UserID, &share.Permission, &share.CreatedAt, &share.CreatedBy,
            &share.User.ID, &share.User.Email, &share.User.Name, &share.User.AvatarURL, &share.User.Provider,
            &share.User.ProviderUserID, &share.User.CreatedAt, &share.User.UpdatedAt, &share.User.LastLoginAt,
        )
        if err != nil {
            return nil, err
        }
        shares = append(shares, share)
    }

    return shares, nil
}

// DeleteDocumentShare deletes a share
func (s *PostgresStore) DeleteDocumentShare(ctx context.Context, documentID, userID uuid.UUID) error {
    _, err := s.db.ExecContext(ctx, "DELETE FROM document_shares WHERE document_id = $1 AND user_id = $2", documentID, userID)
    return err
}

// CanViewDocument checks if user can view document (owner OR has any share)
func (s *PostgresStore) CanViewDocument(ctx context.Context, documentID, userID uuid.UUID) (bool, error) {
    query := `
        SELECT EXISTS(
            SELECT 1 FROM documents WHERE id = $1 AND owner_id = $2
            UNION
            SELECT 1 FROM document_shares WHERE document_id = $1 AND user_id = $2
        )
    `

    var canView bool
    err := s.db.QueryRowContext(ctx, query, documentID, userID).Scan(&canView)
    return canView, err
}

// CanEditDocument checks if user can edit document (owner OR has edit share)
func (s *PostgresStore) CanEditDocument(ctx context.Context, documentID, userID uuid.UUID) (bool, error) {
    query := `
        SELECT EXISTS(
            SELECT 1 FROM documents WHERE id = $1 AND owner_id = $2
            UNION
            SELECT 1 FROM document_shares WHERE document_id = $1 AND user_id = $2 AND permission = 'edit'
        )
    `

    var canEdit bool
    err := s.db.QueryRowContext(ctx, query, documentID, userID).Scan(&canEdit)
    return canEdit, err
}

// IsDocumentOwner checks if user is the owner
func (s *PostgresStore) IsDocumentOwner(ctx context.Context, documentID, userID uuid.UUID) (bool, error) {
    query := `SELECT owner_id = $2 FROM documents WHERE id = $1`

    var isOwner bool
    err := s.db.QueryRowContext(ctx, query, documentID, userID).Scan(&isOwner)
    if err == sql.ErrNoRows {
        return false, nil
    }
    return isOwner, err
}
func (s *PostgresStore) GenerateShareToken(ctx context.Context, documentID uuid.UUID, permission string) (string, error) {
	// Generate random 32-byte token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Update document with share token
	query := `
		UPDATE documents
		SET share_token = $1, is_public = true, updated_at = NOW()
		WHERE id = $2
		RETURNING share_token
	`

	var shareToken string
	err := s.db.QueryRowContext(ctx, query, token, documentID).Scan(&shareToken)
	if err != nil {
		return "", fmt.Errorf("failed to set share token: %w", err)
	}

	return shareToken, nil
}

// ValidateShareToken checks if a share token is valid for a document
func (s *PostgresStore) ValidateShareToken(ctx context.Context, documentID uuid.UUID, token string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM documents
			WHERE id = $1 AND share_token = $2 AND is_public = true
		)
	`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, documentID, token).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to validate share token: %w", err)
	}

	return exists, nil
}

// RevokeShareToken removes the public share link from a document
func (s *PostgresStore) RevokeShareToken(ctx context.Context, documentID uuid.UUID) error {
	query := `
		UPDATE documents
		SET share_token = NULL, is_public = false, updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, documentID)
	if err != nil {
		return fmt.Errorf("failed to revoke share token: %w", err)
	}

	return nil
}

// GetShareToken retrieves the current share token for a document (if exists)
func (s *PostgresStore) GetShareToken(ctx context.Context, documentID uuid.UUID) (string, bool, error) {
	query := `
		SELECT share_token FROM documents
		WHERE id = $1 AND is_public = true AND share_token IS NOT NULL
	`

	var token string
	err := s.db.QueryRowContext(ctx, query, documentID).Scan(&token)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to get share token: %w", err)
	}

	return token, true, nil
}