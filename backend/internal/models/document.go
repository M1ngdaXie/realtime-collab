package models

import (
	"time"

	"github.com/google/uuid"
)

type DocumentType string

const (
	DocumentTypeEditor DocumentType = "editor"
	DocumentTypeKanban DocumentType = "kanban"
)

type Document struct {
    ID        uuid.UUID     `json:"id"`
    Name      string        `json:"name"`
    Type      DocumentType  `json:"type"`
    YjsState  []byte        `json:"-"`
    OwnerID   *uuid.UUID    `json:"owner_id"` // NEW
    Is_Public bool          `json:"is_public"`
    CreatedAt time.Time     `json:"created_at"`
    UpdatedAt time.Time     `json:"updated_at"`
}


type CreateDocumentRequest struct {
  	Name string       `json:"name" binding:"required"`
  	Type DocumentType `json:"type" binding:"required"`
  }

  type UpdateStateRequest struct {
  	State []byte `json:"state" binding:"required"`
  }

  type DocumentListResponse struct {
  	Documents []Document `json:"documents"`
  	Total     int        `json:"total"`
  }

