package models

import (
	"time"

	"github.com/google/uuid"
)

type DocumentShare struct {
	ID         uuid.UUID  `json:"id"`
	DocumentID uuid.UUID  `json:"document_id"`
	UserID     uuid.UUID  `json:"user_id"`
	Permission string     `json:"permission"` // "view" or "edit"
	CreatedAt  time.Time  `json:"created_at"`
	CreatedBy  *uuid.UUID `json:"created_by"`
}

type CreateShareRequest struct {
	UserEmail  string `json:"user_email" binding:"required"`
	Permission string `json:"permission" binding:"required,oneof=view edit"`
}

type ShareListResponse struct {
	Shares []DocumentShareWithUser `json:"shares"`
}

type DocumentShareWithUser struct {
	DocumentShare
	User User `json:"user"`
}

// PermissionResponse represents the user's permission level for a document
type PermissionResponse struct {
	Permission string `json:"permission"` // "view" or "edit"
	Role       string `json:"role"`       // "owner", "editor", or "viewer"
}
