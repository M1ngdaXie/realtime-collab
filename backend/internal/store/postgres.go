package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type Store struct{
	db *sql.DB
}

func NewStore(databaseUrl string) (*Store, error) {
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
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
  	return s.db.Close()
  }

func (s *Store) CreateDocument(name string, docType models.DocumentType) (*models.Document, error) {
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
  func (s *Store) GetDocument(id uuid.UUID) (*models.Document, error) {
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
 func (s *Store) ListDocuments() ([]models.Document, error) {
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

 func (s *Store) UpdateDocumentState(id uuid.UUID, state []byte) error {
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

 func (s *Store) DeleteDocument(id uuid.UUID) error {
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

