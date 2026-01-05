package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TestData holds UUIDs for seeded test data
type TestData struct {
	// Users
	AliceID   uuid.UUID
	BobID     uuid.UUID
	CharlieID uuid.UUID

	// Documents
	AlicePrivateDoc uuid.UUID // Alice's private editor document
	AlicePublicDoc  uuid.UUID // Alice's public kanban document with share token
	BobSharedView   uuid.UUID // Bob's document shared with Alice (view)
	BobSharedEdit   uuid.UUID // Bob's document shared with Alice (edit)
	CharlieDoc      uuid.UUID // Charlie's private document

	// Share token for AlicePublicDoc
	PublicShareToken string
}

// SetupTestDB creates a test database and runs all migrations
func SetupTestDB(t *testing.T) (*PostgresStore, func()) {
	t.Helper()

	// Use environment variable or default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://collab:collab123@localhost:5432/postgres?sslmode=disable"
	}

	// Connect to postgres database (not test db yet)
	masterDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer masterDB.Close()

	// Drop and recreate test database
	testDBName := "collaboration_test"
	_, err = masterDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
	if err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}

	_, err = masterDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to test database
	testDBURL := fmt.Sprintf("postgres://collab:collab123@localhost:5432/%s?sslmode=disable", testDBName)
	store, err := NewPostgresStore(testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	scriptsDir := filepath.Join("..", "..", "scripts")
	migrations := []string{
		"init.sql",
		"001_add_users_and_sessions.sql",
		"002_add_document_shares.sql",
		"003_add_public_sharing.sql",
	}

	for _, migration := range migrations {
		migrationPath := filepath.Join(scriptsDir, migration)
		content, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("Failed to read migration %s: %v", migration, err)
		}

		_, err = store.db.Exec(string(content))
		if err != nil {
			t.Fatalf("Failed to execute migration %s: %v", migration, err)
		}
	}

	// Cleanup function
	cleanup := func() {
		store.Close()
		masterDB, _ := sql.Open("postgres", dbURL)
		if masterDB != nil {
			masterDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
			masterDB.Close()
		}
	}

	return store, cleanup
}

// TruncateAllTables removes all data for test isolation
func TruncateAllTables(ctx context.Context, store *PostgresStore) error {
	tables := []string{
		"document_updates",
		"document_shares",
		"sessions",
		"documents",
		"users",
	}

	for _, table := range tables {
		_, err := store.db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	return nil
}

// SeedTestData creates common test fixtures
func SeedTestData(ctx context.Context, store *PostgresStore) (*TestData, error) {
	data := &TestData{}

	// Create 3 test users
	alice, err := store.UpsertUser(ctx, "google", "alice123", "alice@test.com", "Alice Wonderland", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create alice: %w", err)
	}
	data.AliceID = alice.ID

	bob, err := store.UpsertUser(ctx, "github", "bob456", "bob@test.com", "Bob Builder", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create bob: %w", err)
	}
	data.BobID = bob.ID

	charlie, err := store.UpsertUser(ctx, "google", "charlie789", "charlie@test.com", "Charlie Chaplin", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create charlie: %w", err)
	}
	data.CharlieID = charlie.ID

	// Create documents
	// 1. Alice's private editor document
	doc1, err := store.CreateDocumentWithOwner("Alice Private Doc", models.DocumentTypeEditor, &alice.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create alice private doc: %w", err)
	}
	data.AlicePrivateDoc = doc1.ID

	// 2. Alice's public kanban document
	doc2, err := store.CreateDocumentWithOwner("Alice Public Kanban", models.DocumentTypeKanban, &alice.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create alice public doc: %w", err)
	}
	data.AlicePublicDoc = doc2.ID

	// Generate share token for Alice's public doc
	token, err := store.GenerateShareToken(ctx, doc2.ID, "view")
	if err != nil {
		return nil, fmt.Errorf("failed to generate share token: %w", err)
	}
	data.PublicShareToken = token

	// 3. Bob's document shared with Alice (view permission)
	doc3, err := store.CreateDocumentWithOwner("Bob Shared View", models.DocumentTypeEditor, &bob.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bob shared view doc: %w", err)
	}
	data.BobSharedView = doc3.ID
	_, _, err = store.CreateDocumentShare(ctx, doc3.ID, alice.ID, "view", &bob.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create view share: %w", err)
	}

	// 4. Bob's document shared with Alice (edit permission)
	doc4, err := store.CreateDocumentWithOwner("Bob Shared Edit", models.DocumentTypeEditor, &bob.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bob shared edit doc: %w", err)
	}
	data.BobSharedEdit = doc4.ID
	_, _, err = store.CreateDocumentShare(ctx, doc4.ID, alice.ID, "edit", &bob.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create edit share: %w", err)
	}

	// 5. Charlie's private document
	doc5, err := store.CreateDocumentWithOwner("Charlie Private", models.DocumentTypeEditor, &charlie.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create charlie doc: %w", err)
	}
	data.CharlieDoc = doc5.ID

	return data, nil
}

// GenerateTestJWT creates a valid JWT for testing
func GenerateTestJWT(userID uuid.UUID, secret string) (string, error) {
	expiresIn := 1 * time.Hour

	claims := jwt.MapClaims{
		"sub": userID.String(), // Subject claim for user ID (matches auth.ValidateJWT)
		"exp": time.Now().Add(expiresIn).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
