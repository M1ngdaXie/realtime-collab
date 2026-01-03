package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Store interface defines all database operations
type Store interface {
    // Document operations
    CreateDocument(name string, docType models.DocumentType) (*models.Document, error)
    CreateDocumentWithOwner(name string, docType models.DocumentType, ownerID *uuid.UUID) (*models.Document, error)  // ADD THIS
    GetDocument(id uuid.UUID) (*models.Document, error)
    ListDocuments() ([]models.Document, error)
    ListUserDocuments(ctx context.Context, userID uuid.UUID) ([]models.Document, error)  // ADD THIS
    UpdateDocumentState(id uuid.UUID, state []byte) error
    DeleteDocument(id uuid.UUID) error
    
    // User operations
    UpsertUser(ctx context.Context, provider, providerUserID, email, name string, avatarURL *string) (*models.User, error)
    GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
    GetUserByEmail(ctx context.Context, email string) (*models.User, error)
    
    // Session operations
    CreateSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, token string, expiresAt time.Time, userAgent, ipAddress *string) (*models.Session, error)
    GetSessionByToken(ctx context.Context, token string) (*models.Session, error)
    DeleteSession(ctx context.Context, token string) error
    CleanupExpiredSessions(ctx context.Context) error
    
    // Share operations
    CreateDocumentShare(ctx context.Context, documentID, userID uuid.UUID, permission string, createdBy *uuid.UUID) (*models.DocumentShare, error)
    ListDocumentShares(ctx context.Context, documentID uuid.UUID) ([]models.DocumentShareWithUser, error)
    DeleteDocumentShare(ctx context.Context, documentID, userID uuid.UUID) error
    CanViewDocument(ctx context.Context, documentID, userID uuid.UUID) (bool, error)
    CanEditDocument(ctx context.Context, documentID, userID uuid.UUID) (bool, error)
    IsDocumentOwner(ctx context.Context, documentID, userID uuid.UUID) (bool, error)
	GenerateShareToken(ctx context.Context, documentID uuid.UUID, permission string) (string, error)
	ValidateShareToken(ctx context.Context, documentID uuid.UUID, token string) (bool, error)
	RevokeShareToken(ctx context.Context, documentID uuid.UUID) error
	GetShareToken(ctx context.Context, documentID uuid.UUID) (string, bool, error)

    Close() error
}


type PostgresStore struct {
    db *sql.DB
}

func NewPostgresStore(databaseUrl string) (*PostgresStore, error) {
	db, error := sql.Open("postgres", databaseUrl)
	if error != nil {
		return nil, error
	}
	if err := db.Ping(); err != nil {
  		return nil, fmt.Errorf("failed to ping database: %w", err)
  	}
	db.SetMaxOpenConns(25)
  	db.SetMaxIdleConns(5)
  	db.SetConnMaxLifetime(5 * time.Minute)
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
  	return s.db.Close()
  }

func (s *PostgresStore) CreateDocument(name string, docType models.DocumentType) (*models.Document, error) {
	doc := &models.Document{
		ID:        uuid.New(),
		Name:      name,
		Type:      docType,
		YjsState:  []byte{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	query := `
  		INSERT INTO documents (id, name, type, created_at, updated_at)
  		VALUES ($1, $2, $3, $4, $5)
  		RETURNING id, name, type, created_at, updated_at
	`
	err := s.db.QueryRow(query,
		doc.ID, 
		doc.Name, 
		doc.Type, 
		doc.CreatedAt, 
		doc.UpdatedAt,

	).Scan(&doc.ID, &doc.Name, &doc.Type, &doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}
	return doc, nil
}

 // GetDocument retrieves a document by ID
  func (s *PostgresStore) GetDocument(id uuid.UUID) (*models.Document, error) {
  	doc := &models.Document{}

  	query := `
  		SELECT id, name, type, yjs_state, created_at, updated_at
  		FROM documents
  		WHERE id = $1
  	`

  	err := s.db.QueryRow(query, id).Scan(
  		&doc.ID,
  		&doc.Name,
  		&doc.Type,
  		&doc.YjsState,
  		&doc.CreatedAt,
  		&doc.UpdatedAt,
  	)

  	if err == sql.ErrNoRows {
  		return nil, fmt.Errorf("document not found")
  	}
  	if err != nil {
  		return nil, fmt.Errorf("failed to get document: %w", err)
  	}

  	return doc, nil
  }


 // ListDocuments retrieves all documents
 func (s *PostgresStore) ListDocuments() ([]models.Document, error) {
  	query := `
  		SELECT id, name, type, created_at, updated_at
  		FROM documents
  		ORDER BY created_at DESC
  	`

  	rows, err := s.db.Query(query)
  	if err != nil {
  		return nil, fmt.Errorf("failed to list documents: %w", err)
  	}
  	defer rows.Close()

  	var documents []models.Document
  	for rows.Next() {
  		var doc models.Document
  		err := rows.Scan(&doc.ID, &doc.Name, &doc.Type, &doc.CreatedAt, &doc.UpdatedAt)
  		if err != nil {
  			return nil, fmt.Errorf("failed to scan document: %w", err)
  		}
  		documents = append(documents, doc)
  	}

  	return documents, nil
  }

 func (s *PostgresStore) UpdateDocumentState(id uuid.UUID, state []byte) error {
  	query := `
  		UPDATE documents
  		SET yjs_state = $1, updated_at = $2
  		WHERE id = $3
  	`

  	result, err := s.db.Exec(query, state, time.Now(), id)
  	if err != nil {
  		return fmt.Errorf("failed to update document state: %w", err)
  	}

  	rowsAffected, err := result.RowsAffected()
  	if err != nil {
  		return fmt.Errorf("failed to get rows affected: %w", err)
  	}

  	if rowsAffected == 0 {
  		return fmt.Errorf("document not found")
  	}

  	return nil
  }

 func (s *PostgresStore) DeleteDocument(id uuid.UUID) error {
  	query := `DELETE FROM documents WHERE id = $1`

  	result, err := s.db.Exec(query, id)
  	if err != nil {
  		return fmt.Errorf("failed to delete document: %w", err)
  	}

  	rowsAffected, err := result.RowsAffected()
  	if err != nil {
  		return fmt.Errorf("failed to get rows affected: %w", err)
  	}

  	if rowsAffected == 0 {
  		return fmt.Errorf("document not found")
  	}

  	return nil
  }

// CreateDocumentWithOwner creates a new document with owner
func (s *PostgresStore) CreateDocumentWithOwner(name string, docType models.DocumentType, ownerID *uuid.UUID) (*models.Document, error) {
    // 1. 检查 docType 是否为空，或者是否合法 (防止 check constraint 报错)
    if docType == "" {
        docType = models.DocumentTypeEditor // Default to editor instead of invalid "text"
    }

    // Validate that docType is one of the allowed values
    if docType != models.DocumentTypeEditor && docType != models.DocumentTypeKanban {
        return nil, fmt.Errorf("invalid document type: %s (must be 'editor' or 'kanban')", docType)
    }

    doc := &models.Document{
        ID:        uuid.New(),
        Name:      name,
        Type:      docType,
        YjsState:  []byte{}, // 这里初始化了空字节
        OwnerID:   ownerID,
        Is_Public:  false,    // 显式设置默认值
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // 2. 补全了 yjs_state 和 is_public
    query := `
        INSERT INTO documents (id, name, type, owner_id, yjs_state, is_public, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, name, type, owner_id, yjs_state, is_public, created_at, updated_at
    `
    
    // 3. Scan 的时候也要对应加上
    err := s.db.QueryRow(query,
        doc.ID,
        doc.Name,
        doc.Type,
        doc.OwnerID,
        doc.YjsState, // $5
        doc.Is_Public, // $6
        doc.CreatedAt,
        doc.UpdatedAt,
    ).Scan(
        &doc.ID, 
        &doc.Name, 
        &doc.Type, 
        &doc.OwnerID, 
        &doc.YjsState, // Scan 回来
        &doc.Is_Public, // Scan 回来
        &doc.CreatedAt, 
        &doc.UpdatedAt,
    )

    if err != nil {
        return nil, fmt.Errorf("failed to create document: %w", err)
    }
    return doc, nil
}

// ListUserDocuments lists documents owned by or shared with a user
func (s *PostgresStore) ListUserDocuments(ctx context.Context, userID uuid.UUID) ([]models.Document, error) {
	query := `
		SELECT DISTINCT d.id, d.name, d.type, d.owner_id, d.created_at, d.updated_at
		FROM documents d
		LEFT JOIN document_shares ds ON d.id = ds.document_id
		WHERE d.owner_id = $1 OR ds.user_id = $1
		ORDER BY d.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user documents: %w", err)
	}
	defer rows.Close()

	var documents []models.Document
	for rows.Next() {
		var doc models.Document
		err := rows.Scan(&doc.ID, &doc.Name, &doc.Type, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		documents = append(documents, doc)
	}

	return documents, nil
}
