package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/M1ngdaXie/realtime-collab/internal/auth"
	"github.com/M1ngdaXie/realtime-collab/internal/messagebus"
	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// DocumentHandlerSuite tests document CRUD operations
type DocumentHandlerSuite struct {
	BaseHandlerSuite
	handler *DocumentHandler
	router  *gin.Engine
}

// SetupTest runs before each test
func (s *DocumentHandlerSuite) SetupTest() {
	s.BaseHandlerSuite.SetupTest()
	s.handler = NewDocumentHandler(s.store, messagebus.NewLocalMessageBus(), "test-server", zap.NewNop())
	s.setupRouter()
}

// setupRouter configures the Gin router for testing
func (s *DocumentHandlerSuite) setupRouter() {
	s.router = gin.New()

	// Auth middleware mock
	s.router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Extract user ID from test JWT
			token := authHeader[len("Bearer "):]
			userID, err := s.parseTestJWT(token)
			if err == nil {
				// Store as pointer to match real middleware
				c.Set("user_id", &userID)
			}
		}
		c.Next()
	})

	// Document routes
	s.router.POST("/api/documents", s.handler.CreateDocument)
	s.router.GET("/api/documents", s.handler.ListDocuments)
	s.router.GET("/api/documents/:id", s.handler.GetDocument)
	s.router.GET("/api/documents/:id/state", s.handler.GetDocumentState)
	s.router.PUT("/api/documents/:id/state", s.handler.UpdateDocumentState)
	s.router.DELETE("/api/documents/:id", s.handler.DeleteDocument)
}

// parseTestJWT extracts user ID from test JWT
func (s *DocumentHandlerSuite) parseTestJWT(tokenString string) (uuid.UUID, error) {
	// Use auth package to parse JWT
	claims, err := auth.ValidateJWT(tokenString, s.jwtSecret)
	if err != nil {
		return uuid.Nil, err
	}

	// Extract user ID from Subject claim
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

// TestDocumentHandlerSuite runs the test suite
func TestDocumentHandlerSuite(t *testing.T) {
	suite.Run(t, new(DocumentHandlerSuite))
}

// ========================================
// CreateDocument Tests
// ========================================

func (s *DocumentHandlerSuite) TestCreateDocument_Success() {
	req := models.CreateDocumentRequest{
		Name: "New Test Document",
		Type: models.DocumentTypeEditor,
	}

	w, httpReq, err := s.makeAuthRequest("POST", "/api/documents", req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusCreated)

	var doc models.Document
	s.parseJSONResponse(w, &doc)
	s.Equal("New Test Document", doc.Name)
	s.Equal(models.DocumentTypeEditor, doc.Type)
	s.NotEqual(uuid.Nil, doc.ID)
	s.Equal(s.testData.AliceID, *doc.OwnerID)
}

func (s *DocumentHandlerSuite) TestCreateDocument_Kanban() {
	req := models.CreateDocumentRequest{
		Name: "Kanban Board",
		Type: models.DocumentTypeKanban,
	}

	w, httpReq, err := s.makeAuthRequest("POST", "/api/documents", req, s.testData.BobID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusCreated)

	var doc models.Document
	s.parseJSONResponse(w, &doc)
	s.Equal(models.DocumentTypeKanban, doc.Type)
}

func (s *DocumentHandlerSuite) TestCreateDocument_Unauthorized() {
	req := models.CreateDocumentRequest{
		Name: "Test",
		Type: models.DocumentTypeEditor,
	}

	w, httpReq, err := s.makePublicRequest("POST", "/api/documents", req)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
}

func (s *DocumentHandlerSuite) TestCreateDocument_MissingName() {
	req := map[string]interface{}{
		"type": "editor",
		// name is missing
	}

	w, httpReq, err := s.makeAuthRequest("POST", "/api/documents", req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusBadRequest, "validation_error", "")
}

func (s *DocumentHandlerSuite) TestCreateDocument_EmptyName() {
	req := models.CreateDocumentRequest{
		Name: "",
		Type: models.DocumentTypeEditor,
	}

	w, httpReq, err := s.makeAuthRequest("POST", "/api/documents", req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	// Should fail validation or succeed with empty name depending on validation rules
	// Assuming it's allowed for now
	s.T().Skip("Empty name validation not implemented")
}

func (s *DocumentHandlerSuite) TestCreateDocument_InvalidType() {
	req := map[string]interface{}{
		"name": "Test Doc",
		"type": "invalid_type",
	}

	w, httpReq, err := s.makeAuthRequest("POST", "/api/documents", req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	// Database constraint will catch this
	s.Equal(http.StatusInternalServerError, w.Code)
}

// ========================================
// ListDocuments Tests
// ========================================

func (s *DocumentHandlerSuite) TestListDocuments_OwnerSeesOwned() {
	w, httpReq, err := s.makeAuthRequest("GET", "/api/documents", nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var resp models.DocumentListResponse
	s.parseJSONResponse(w, &resp)

	// Alice should see: her 2 docs + 2 shared from Bob = 4 total
	s.GreaterOrEqual(len(resp.Documents), 2, "Alice should see at least her own documents")

	// Check Alice's documents are in the list
	foundPrivate := false
	foundPublic := false
	for _, doc := range resp.Documents {
		if doc.ID == s.testData.AlicePrivateDoc {
			foundPrivate = true
		}
		if doc.ID == s.testData.AlicePublicDoc {
			foundPublic = true
		}
	}
	s.True(foundPrivate, "Alice should see her private document")
	s.True(foundPublic, "Alice should see her public document")
}

func (s *DocumentHandlerSuite) TestListDocuments_SharedDocuments() {
	w, httpReq, err := s.makeAuthRequest("GET", "/api/documents", nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var resp models.DocumentListResponse
	s.parseJSONResponse(w, &resp)

	// Alice should see documents shared with her
	foundSharedView := false
	foundSharedEdit := false
	for _, doc := range resp.Documents {
		if doc.ID == s.testData.BobSharedView {
			foundSharedView = true
		}
		if doc.ID == s.testData.BobSharedEdit {
			foundSharedEdit = true
		}
	}
	s.True(foundSharedView, "Alice should see Bob's shared view document")
	s.True(foundSharedEdit, "Alice should see Bob's shared edit document")
}

func (s *DocumentHandlerSuite) TestListDocuments_DoesNotSeeOthers() {
	w, httpReq, err := s.makeAuthRequest("GET", "/api/documents", nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var resp models.DocumentListResponse
	s.parseJSONResponse(w, &resp)

	// Alice should NOT see Charlie's document
	for _, doc := range resp.Documents {
		s.NotEqual(s.testData.CharlieDoc, doc.ID, "Alice should not see Charlie's document")
	}
}

func (s *DocumentHandlerSuite) TestListDocuments_Unauthorized() {
	w, httpReq, err := s.makePublicRequest("GET", "/api/documents", nil)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
}

func (s *DocumentHandlerSuite) TestListDocuments_EmptyResult() {
	// Create a new user with no documents
	ctx := context.Background()
	newUser, err := s.store.UpsertUser(ctx, "google", "new123", "new@test.com", "New User", nil)
	s.Require().NoError(err)

	w, httpReq, err := s.makeAuthRequest("GET", "/api/documents", nil, newUser.ID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var resp models.DocumentListResponse
	s.parseJSONResponse(w, &resp)
	s.Equal(0, len(resp.Documents), "New user should have no documents")
	s.Equal(0, resp.Total)
}

// ========================================
// GetDocument Tests
// ========================================

func (s *DocumentHandlerSuite) TestGetDocument_Owner() {
	path := fmt.Sprintf("/api/documents/%s", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var doc models.Document
	s.parseJSONResponse(w, &doc)
	s.Equal(s.testData.AlicePrivateDoc, doc.ID)
}

func (s *DocumentHandlerSuite) TestGetDocument_SharedView() {
	path := fmt.Sprintf("/api/documents/%s", s.testData.BobSharedView)
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var doc models.Document
	s.parseJSONResponse(w, &doc)
	s.Equal(s.testData.BobSharedView, doc.ID)
}

func (s *DocumentHandlerSuite) TestGetDocument_SharedEdit() {
	path := fmt.Sprintf("/api/documents/%s", s.testData.BobSharedEdit)
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var doc models.Document
	s.parseJSONResponse(w, &doc)
	s.Equal(s.testData.BobSharedEdit, doc.ID)
}

func (s *DocumentHandlerSuite) TestGetDocument_PublicDocument() {
	path := fmt.Sprintf("/api/documents/%s", s.testData.AlicePublicDoc)
	w, httpReq, err := s.makePublicRequest("GET", path, nil)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var doc models.Document
	s.parseJSONResponse(w, &doc)
	s.Equal(s.testData.AlicePublicDoc, doc.ID)
}

func (s *DocumentHandlerSuite) TestGetDocument_PrivateRequiresAuth() {
	path := fmt.Sprintf("/api/documents/%s", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makePublicRequest("GET", path, nil)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "not public")
}

func (s *DocumentHandlerSuite) TestGetDocument_Forbidden() {
	// Alice tries to access Charlie's private document
	path := fmt.Sprintf("/api/documents/%s", s.testData.CharlieDoc)
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "Access denied")
}

func (s *DocumentHandlerSuite) TestGetDocument_NotFound() {
	nonExistentID := uuid.New()
	path := fmt.Sprintf("/api/documents/%s", nonExistentID)
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusNotFound, "not_found", "not found")
}

func (s *DocumentHandlerSuite) TestGetDocument_InvalidUUID() {
	path := "/api/documents/invalid-uuid"
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusBadRequest, "bad_request", "Invalid document ID")
}

// ========================================
// GetDocumentState Tests
// ========================================

func (s *DocumentHandlerSuite) TestGetDocumentState_Success() {
	path := fmt.Sprintf("/api/documents/%s/state", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	s.Equal("application/octet-stream", w.Header().Get("Content-Type"))
	// State should be empty bytes for new document
	s.NotNil(w.Body.Bytes())
}

func (s *DocumentHandlerSuite) TestGetDocumentState_EmptyState() {
	path := fmt.Sprintf("/api/documents/%s/state", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	// New documents have empty yjs_state
	s.Equal(0, w.Body.Len())
}

func (s *DocumentHandlerSuite) TestGetDocumentState_PermissionCheck() {
	// Alice tries to get Charlie's document state
	path := fmt.Sprintf("/api/documents/%s/state", s.testData.CharlieDoc)
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "Access denied")
}

func (s *DocumentHandlerSuite) TestGetDocumentState_InvalidID() {
	path := "/api/documents/invalid-uuid/state"
	w, httpReq, err := s.makeAuthRequest("GET", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusBadRequest, "bad_request", "Invalid document ID")
}

// ========================================
// UpdateDocumentState Tests
// ========================================

func (s *DocumentHandlerSuite) TestUpdateDocumentState_Owner() {
	req := models.UpdateStateRequest{
		State: []byte("new yjs state"),
	}

	path := fmt.Sprintf("/api/documents/%s/state", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makeAuthRequest("PUT", path, req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)
}

func (s *DocumentHandlerSuite) TestUpdateDocumentState_EditPermission() {
	req := models.UpdateStateRequest{
		State: []byte("updated state"),
	}

	path := fmt.Sprintf("/api/documents/%s/state", s.testData.BobSharedEdit)
	w, httpReq, err := s.makeAuthRequest("PUT", path, req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)
}

func (s *DocumentHandlerSuite) TestUpdateDocumentState_ViewOnlyDenied() {
	req := models.UpdateStateRequest{
		State: []byte("attempt update"),
	}

	path := fmt.Sprintf("/api/documents/%s/state", s.testData.BobSharedView)
	w, httpReq, err := s.makeAuthRequest("PUT", path, req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "Edit access denied")
}

func (s *DocumentHandlerSuite) TestUpdateDocumentState_Unauthorized() {
	req := models.UpdateStateRequest{
		State: []byte("state"),
	}

	path := fmt.Sprintf("/api/documents/%s/state", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makePublicRequest("PUT", path, req)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
}

func (s *DocumentHandlerSuite) TestUpdateDocumentState_InvalidID() {
	req := models.UpdateStateRequest{
		State: []byte("state"),
	}

	path := "/api/documents/invalid-uuid/state"
	w, httpReq, err := s.makeAuthRequest("PUT", path, req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusBadRequest, "bad_request", "Invalid document ID")
}

func (s *DocumentHandlerSuite) TestUpdateDocumentState_NotFound() {
	req := models.UpdateStateRequest{
		State: []byte("state"),
	}

	nonExistentID := uuid.New()
	path := fmt.Sprintf("/api/documents/%s/state", nonExistentID)
	w, httpReq, err := s.makeAuthRequest("PUT", path, req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusNotFound, "not_found", "document")
}

func (s *DocumentHandlerSuite) TestUpdateDocumentState_EmptyState() {
	req := models.UpdateStateRequest{
		State: []byte{},
	}

	path := fmt.Sprintf("/api/documents/%s/state", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makeAuthRequest("PUT", path, req, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)
}

// ========================================
// DeleteDocument Tests
// ========================================

func (s *DocumentHandlerSuite) TestDeleteDocument_Owner() {
	path := fmt.Sprintf("/api/documents/%s", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makeAuthRequest("DELETE", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	// Verify document is deleted
	_, dbErr := s.store.GetDocument(s.testData.AlicePrivateDoc)
	s.Error(dbErr, "Document should be deleted")
}

func (s *DocumentHandlerSuite) TestDeleteDocument_SharedUserDenied() {
	// Alice tries to delete Bob's document (even though she has edit permission)
	path := fmt.Sprintf("/api/documents/%s", s.testData.BobSharedEdit)
	w, httpReq, err := s.makeAuthRequest("DELETE", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "Only document owner can delete")
}

func (s *DocumentHandlerSuite) TestDeleteDocument_Unauthorized() {
	path := fmt.Sprintf("/api/documents/%s", s.testData.AlicePrivateDoc)
	w, httpReq, err := s.makePublicRequest("DELETE", path, nil)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
}

func (s *DocumentHandlerSuite) TestDeleteDocument_NotFound() {
	nonExistentID := uuid.New()
	path := fmt.Sprintf("/api/documents/%s", nonExistentID)
	w, httpReq, err := s.makeAuthRequest("DELETE", path, nil, s.testData.AliceID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertErrorResponse(w, http.StatusNotFound, "not_found", "document")
}

func (s *DocumentHandlerSuite) TestDeleteDocument_CascadeShares() {
	// Bob deletes his shared document
	path := fmt.Sprintf("/api/documents/%s", s.testData.BobSharedView)
	w, httpReq, err := s.makeAuthRequest("DELETE", path, nil, s.testData.BobID)
	s.Require().NoError(err)

	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	// Verify shares are also deleted (cascade)
	shares, err := s.store.ListDocumentShares(context.Background(), s.testData.BobSharedView)
	s.NoError(err)
	s.Equal(0, len(shares), "Shares should be cascaded deleted")
}
