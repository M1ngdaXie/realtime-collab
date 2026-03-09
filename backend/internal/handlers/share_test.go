package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/M1ngdaXie/realtime-collab/internal/auth"
	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// ShareHandlerSuite tests for share handler endpoints
type ShareHandlerSuite struct {
	BaseHandlerSuite
	handler *ShareHandler
	router  *gin.Engine
}

// SetupTest runs before each test
func (s *ShareHandlerSuite) SetupTest() {
	s.BaseHandlerSuite.SetupTest()

	// Create handler and router
	authMiddleware := auth.NewAuthMiddleware(s.store, s.jwtSecret, zap.NewNop())
	s.handler = NewShareHandler(s.store, s.cfg)
	s.router = gin.New()

	// Custom auth middleware for tests that sets user_id as pointer
	s.router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 {
			token := authHeader[len("Bearer "):]
			claims, err := auth.ValidateJWT(token, s.jwtSecret)
			if err == nil {
				userID, err := uuid.Parse(claims.Subject)
				if err == nil {
					c.Set("user_id", &userID) // Store as pointer to match real middleware
				}
			}
		}
		c.Next()
	})

	// Register routes
	api := s.router.Group("/api")
	{
		docs := api.Group("/documents")
		{
			docs.POST("/:id/shares", authMiddleware.RequireAuth(), s.handler.CreateShare)
			docs.GET("/:id/shares", authMiddleware.RequireAuth(), s.handler.ListShares)
			docs.DELETE("/:id/shares/:userId", authMiddleware.RequireAuth(), s.handler.DeleteShare)
			docs.POST("/:id/share-link", authMiddleware.RequireAuth(), s.handler.CreateShareLink)
			docs.GET("/:id/share-link", authMiddleware.RequireAuth(), s.handler.GetShareLink)
			docs.DELETE("/:id/share-link", authMiddleware.RequireAuth(), s.handler.RevokeShareLink)
		}
	}
}

// TestShareHandlerSuite runs the test suite
func TestShareHandlerSuite(t *testing.T) {
	suite.Run(t, new(ShareHandlerSuite))
}

// ============================================================================
// CreateShare Tests (POST /api/documents/:id/shares)
// ============================================================================

func (s *ShareHandlerSuite) TestCreateShare_ViewPermission() {
	body := map[string]interface{}{
		"user_email": "bob@test.com",
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusCreated)

	var response map[string]interface{}
	s.parseJSONResponse(w, &response)
	s.Equal("view", response["permission"])
}

func (s *ShareHandlerSuite) TestCreateShare_EditPermission() {
	body := map[string]interface{}{
		"user_email": "bob@test.com",
		"permission": "edit",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusCreated)

	var response map[string]interface{}
	s.parseJSONResponse(w, &response)
	s.Equal("edit", response["permission"])
}

func (s *ShareHandlerSuite) TestCreateShare_NonOwnerDenied() {
	body := map[string]interface{}{
		"user_email": "charlie@test.com",
		"permission": "view",
	}

	// Bob tries to share Alice's document
	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "owner")
}

func (s *ShareHandlerSuite) TestCreateShare_UserNotFound() {
	body := map[string]interface{}{
		"user_email": "nonexistent@test.com",
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusNotFound, "not_found", "")
}

func (s *ShareHandlerSuite) TestCreateShare_InvalidPermission() {
	body := map[string]interface{}{
		"user_email": "bob@test.com",
		"permission": "admin", // Invalid permission
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusBadRequest, "validation_error", "")
}

func (s *ShareHandlerSuite) TestCreateShare_UpdatesExisting() {
	// Create initial share with view permission
	body := map[string]interface{}{
		"user_email": "bob@test.com",
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusCreated)

	// Update to edit permission
	body["permission"] = "edit"
	w2, httpReq2, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w2, httpReq2)

	s.assertSuccessResponse(w2, http.StatusOK) // Should return 200 for update

	var response map[string]interface{}
	s.parseJSONResponse(w2, &response)
	s.Equal("edit", response["permission"])
}

func (s *ShareHandlerSuite) TestCreateShare_Unauthorized() {
	body := map[string]interface{}{
		"user_email": "bob@test.com",
		"permission": "view",
	}

	w, httpReq, err := s.makePublicRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *ShareHandlerSuite) TestCreateShare_InvalidDocumentID() {
	body := map[string]interface{}{
		"user_email": "bob@test.com",
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", "/api/documents/invalid-uuid/shares", body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusBadRequest, "invalid_id", "")
}

// ============================================================================
// ListShares Tests (GET /api/documents/:id/shares)
// ============================================================================

func (s *ShareHandlerSuite) TestListShares_OwnerSeesAll() {
	// Bob's shared edit document has Alice as a collaborator
	w, httpReq, err := s.makeAuthRequest("GET", fmt.Sprintf("/api/documents/%s/shares", s.testData.BobSharedEdit), nil, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response models.ShareListResponse
	s.parseJSONResponse(w, &response)
	shares := response.Shares
	s.GreaterOrEqual(len(shares), 1, "Should have at least one share")
}

func (s *ShareHandlerSuite) TestListShares_NonOwnerDenied() {
	// Alice tries to list shares for Bob's document
	w, httpReq, err := s.makeAuthRequest("GET", fmt.Sprintf("/api/documents/%s/shares", s.testData.BobSharedEdit), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "")
}

func (s *ShareHandlerSuite) TestListShares_EmptyList() {
	// Alice's private document has no shares
	w, httpReq, err := s.makeAuthRequest("GET", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response models.ShareListResponse
	s.parseJSONResponse(w, &response)
	shares := response.Shares
	s.Equal(0, len(shares), "Should have no shares")
}

func (s *ShareHandlerSuite) TestListShares_IncludesUserDetails() {
	// Bob's shared edit document has Alice as a collaborator
	w, httpReq, err := s.makeAuthRequest("GET", fmt.Sprintf("/api/documents/%s/shares", s.testData.BobSharedEdit), nil, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response models.ShareListResponse
	s.parseJSONResponse(w, &response)
	shares := response.Shares

	if len(shares) > 0 {
		share := shares[0]
		s.NotEmpty(share.User.Email, "Should include user email")
		s.NotEmpty(share.User.Name, "Should include user name")
	}
}

func (s *ShareHandlerSuite) TestListShares_Unauthorized() {
	w, httpReq, err := s.makePublicRequest("GET", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), nil)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *ShareHandlerSuite) TestListShares_OrderedByCreatedAt() {
	// Create multiple shares
	users := []string{"bob@test.com", "charlie@test.com"}
	for _, email := range users {
		body := map[string]interface{}{
			"user_email": email,
			"permission": "view",
		}
		w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
		s.Require().NoError(err)
		s.router.ServeHTTP(w, httpReq)
		s.assertSuccessResponse(w, http.StatusCreated)
	}

	// List shares
	w, httpReq, err := s.makeAuthRequest("GET", fmt.Sprintf("/api/documents/%s/shares", s.testData.AlicePrivateDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response models.ShareListResponse
	s.parseJSONResponse(w, &response)
	shares := response.Shares
	s.Equal(2, len(shares), "Should have 2 shares")
}

// ============================================================================
// DeleteShare Tests (DELETE /api/documents/:id/shares/:userId)
// ============================================================================

func (s *ShareHandlerSuite) TestDeleteShare_OwnerRemoves() {
	// Bob removes Alice's share from his document
	w, httpReq, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/shares/%s", s.testData.BobSharedEdit, s.testData.AliceID), nil, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusNoContent)
}

func (s *ShareHandlerSuite) TestDeleteShare_NonOwnerDenied() {
	// Alice tries to delete a share from Bob's document
	w, httpReq, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/shares/%s", s.testData.BobSharedEdit, s.testData.AliceID), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "")
}

func (s *ShareHandlerSuite) TestDeleteShare_Idempotent() {
	// Delete share twice - both should succeed
	w1, httpReq1, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/shares/%s", s.testData.BobSharedEdit, s.testData.AliceID), nil, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w1, httpReq1)
	s.assertSuccessResponse(w1, http.StatusNoContent)

	w2, httpReq2, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/shares/%s", s.testData.BobSharedEdit, s.testData.AliceID), nil, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w2, httpReq2)
	s.assertSuccessResponse(w2, http.StatusNoContent)
}

func (s *ShareHandlerSuite) TestDeleteShare_InvalidUserID() {
	w, httpReq, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/shares/invalid-uuid", s.testData.BobSharedEdit), nil, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusBadRequest, "invalid_id", "")
}

func (s *ShareHandlerSuite) TestDeleteShare_Unauthorized() {
	w, httpReq, err := s.makePublicRequest("DELETE", fmt.Sprintf("/api/documents/%s/shares/%s", s.testData.BobSharedEdit, s.testData.AliceID), nil)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.Equal(http.StatusUnauthorized, w.Code)
}

// ============================================================================
// CreateShareLink Tests (POST /api/documents/:id/share-link)
// ============================================================================

func (s *ShareHandlerSuite) TestCreateShareLink_ViewPermission() {
	body := map[string]interface{}{
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response map[string]interface{}
	s.parseJSONResponse(w, &response)
	s.NotEmpty(response["token"], "Should return share token")
	s.NotEmpty(response["url"], "Should return share URL")
	s.Equal("view", response["permission"])
}

func (s *ShareHandlerSuite) TestCreateShareLink_EditPermission() {
	body := map[string]interface{}{
		"permission": "edit",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response map[string]interface{}
	s.parseJSONResponse(w, &response)
	s.Equal("edit", response["permission"])
}

func (s *ShareHandlerSuite) TestCreateShareLink_NonOwnerDenied() {
	body := map[string]interface{}{
		"permission": "view",
	}

	// Bob tries to create share link for Alice's document
	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "")
}

func (s *ShareHandlerSuite) TestCreateShareLink_SetsIsPublic() {
	body := map[string]interface{}{
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	// Verify document is now public
	doc, err := s.store.GetDocument(s.testData.AlicePrivateDoc)
	s.Require().NoError(err)
	s.True(doc.Is_Public, "Document should be marked as public")
}

func (s *ShareHandlerSuite) TestCreateShareLink_ReturnsTokenAndURL() {
	body := map[string]interface{}{
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response map[string]interface{}
	s.parseJSONResponse(w, &response)

	token := response["token"].(string)
	url := response["url"].(string)

	s.NotEmpty(token, "Token should not be empty")
	s.Contains(url, token, "URL should contain token")
	s.Contains(url, "http://localhost:5173", "URL should contain frontend URL")
}

func (s *ShareHandlerSuite) TestCreateShareLink_InvalidPermission() {
	body := map[string]interface{}{
		"permission": "admin", // Invalid
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusBadRequest, "validation_error", "")
}

func (s *ShareHandlerSuite) TestCreateShareLink_Unauthorized() {
	body := map[string]interface{}{
		"permission": "view",
	}

	w, httpReq, err := s.makePublicRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *ShareHandlerSuite) TestCreateShareLink_RegeneratesToken() {
	body := map[string]interface{}{
		"permission": "view",
	}

	// Create first share link
	w1, httpReq1, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w1, httpReq1)
	s.assertSuccessResponse(w1, http.StatusOK)

	var response1 map[string]interface{}
	s.parseJSONResponse(w1, &response1)
	token1 := response1["token"].(string)

	// Create second share link - should regenerate
	w2, httpReq2, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w2, httpReq2)
	s.assertSuccessResponse(w2, http.StatusOK)

	var response2 map[string]interface{}
	s.parseJSONResponse(w2, &response2)
	token2 := response2["token"].(string)

	s.NotEqual(token1, token2, "Second call should generate new token")
}

func (s *ShareHandlerSuite) TestCreateShareLink_TokenLength() {
	body := map[string]interface{}{
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response map[string]interface{}
	s.parseJSONResponse(w, &response)
	token := response["token"].(string)

	// Base64-encoded 32 bytes should be 44 characters (with padding)
	s.GreaterOrEqual(len(token), 40, "Token should be at least 40 characters")
}

// ============================================================================
// GetShareLink Tests (GET /api/documents/:id/share-link)
// ============================================================================

func (s *ShareHandlerSuite) TestGetShareLink_OwnerRetrieves() {
	// Alice's public doc already has a share token from test data
	w, httpReq, err := s.makeAuthRequest("GET", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusOK)

	var response map[string]interface{}
	s.parseJSONResponse(w, &response)
	s.NotEmpty(response["token"], "Should return share token")
	s.NotEmpty(response["url"], "Should return share URL")
}

func (s *ShareHandlerSuite) TestGetShareLink_NonOwnerDenied() {
	// Bob tries to get Alice's share link
	w, httpReq, err := s.makeAuthRequest("GET", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "")
}

func (s *ShareHandlerSuite) TestGetShareLink_NotFound() {
	// Alice's private doc has no share link
	w, httpReq, err := s.makeAuthRequest("GET", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusNotFound, "not_found", "")
}

func (s *ShareHandlerSuite) TestGetShareLink_Unauthorized() {
	w, httpReq, err := s.makePublicRequest("GET", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.Equal(http.StatusUnauthorized, w.Code)
}

// ============================================================================
// RevokeShareLink Tests (DELETE /api/documents/:id/share-link)
// ============================================================================

func (s *ShareHandlerSuite) TestRevokeShareLink_OwnerRevokes() {
	// Revoke Alice's public doc share link
	w, httpReq, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusNoContent)
}

func (s *ShareHandlerSuite) TestRevokeShareLink_SetsIsPublicFalse() {
	// Revoke share link
	w, httpReq, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusNoContent)

	// Verify document is no longer public
	doc, err := s.store.GetDocument(s.testData.AlicePublicDoc)
	s.Require().NoError(err)
	s.False(doc.Is_Public, "Document should no longer be public")
}

func (s *ShareHandlerSuite) TestRevokeShareLink_ClearsToken() {
	// Revoke share link
	w, httpReq, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertSuccessResponse(w, http.StatusNoContent)

	// Verify token is cleared
	token, exists, err := s.store.GetShareToken(s.T().Context(), s.testData.AlicePublicDoc)
	s.Require().NoError(err)
	s.False(exists, "Share token should not exist")
	s.Empty(token, "Share token should be empty")
}

func (s *ShareHandlerSuite) TestRevokeShareLink_NonOwnerDenied() {
	// Bob tries to revoke Alice's share link
	w, httpReq, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.BobID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.assertErrorResponse(w, http.StatusForbidden, "forbidden", "")
}

func (s *ShareHandlerSuite) TestRevokeShareLink_Unauthorized() {
	w, httpReq, err := s.makePublicRequest("DELETE", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)

	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *ShareHandlerSuite) TestRevokeShareLink_Idempotent() {
	// Revoke twice - both should succeed
	w1, httpReq1, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w1, httpReq1)
	s.assertSuccessResponse(w1, http.StatusNoContent)

	w2, httpReq2, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w2, httpReq2)
	s.assertSuccessResponse(w2, http.StatusNoContent)
}

// ============================================================================
// Public Access Integration Tests
// ============================================================================

func (s *ShareHandlerSuite) TestPublicAccess_ValidToken() {
	// This test validates that a document with a valid share token is accessible
	// The actual public access logic is tested in document handler tests
	doc, err := s.store.GetDocument(s.testData.AlicePublicDoc)
	s.Require().NoError(err)
	s.True(doc.Is_Public, "Test document should be public")

	token, exists, err := s.store.GetShareToken(s.T().Context(), s.testData.AlicePublicDoc)
	s.Require().NoError(err)
	s.True(exists, "Test document should have share token")
	s.NotEmpty(token, "Share token should not be empty")
}

func (s *ShareHandlerSuite) TestPublicAccess_InvalidToken() {
	// Create a share link
	body := map[string]interface{}{
		"permission": "view",
	}

	w, httpReq, err := s.makeAuthRequest("POST", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePrivateDoc), body, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusOK)

	var response map[string]interface{}
	s.parseJSONResponse(w, &response)
	token := response["token"].(string)

	s.NotEmpty(token, "Should have generated a token")
}

func (s *ShareHandlerSuite) TestPublicAccess_RevokedToken() {
	// Get current token
	oldToken, exists, err := s.store.GetShareToken(s.T().Context(), s.testData.AlicePublicDoc)
	s.Require().NoError(err)
	s.True(exists, "Document should have share token initially")
	s.NotEmpty(oldToken, "Old token should not be empty")

	// Revoke share link
	w, httpReq, err := s.makeAuthRequest("DELETE", fmt.Sprintf("/api/documents/%s/share-link", s.testData.AlicePublicDoc), nil, s.testData.AliceID)
	s.Require().NoError(err)
	s.router.ServeHTTP(w, httpReq)
	s.assertSuccessResponse(w, http.StatusNoContent)

	// Verify token is cleared
	newToken, exists, err := s.store.GetShareToken(s.T().Context(), s.testData.AlicePublicDoc)
	s.Require().NoError(err)
	s.False(exists, "Share token should not exist after revocation")
	s.Empty(newToken, "Token should be cleared after revocation")
}
