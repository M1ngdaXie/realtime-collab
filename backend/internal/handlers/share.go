package handlers

import (
	"fmt"
	"net/http"
	"os" // Add this

	"github.com/M1ngdaXie/realtime-collab/internal/auth"
	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShareHandler struct {
	store store.Store
}

func NewShareHandler(store store.Store) *ShareHandler {
	return &ShareHandler{store: store}
}

// CreateShare creates a new document share
func (h *ShareHandler) CreateShare(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "Invalid document ID format")
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check ownership", err)
		return
	}
	if !isOwner {
		respondForbidden(c, "Only document owner can share documents")
		return
	}

	var req models.CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithValidationError(c, err)
		return
	}

	// Get user by email
	targetUser, err := h.store.GetUserByEmail(c.Request.Context(), req.UserEmail)
	if err != nil || targetUser == nil {
		respondNotFound(c, "user")
		return
	}

	// Create or update share
	share, isNew, err := h.store.CreateDocumentShare(
		c.Request.Context(),
		documentID,
		targetUser.ID,
		req.Permission,
		userID,
	)
	if err != nil {
		respondInternalError(c, "Failed to create share", err)
		return
	}

	// Return 201 for new share, 200 for updated share
	statusCode := http.StatusOK
	if isNew {
		statusCode = http.StatusCreated
	}
	c.JSON(statusCode, share)
}

// ListShares lists all shares for a document
func (h *ShareHandler) ListShares(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "Invalid document ID format")
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check ownership", err)
		return
	}
	if !isOwner {
		respondForbidden(c, "Only document owner can view shares")
		return
	}

	shares, err := h.store.ListDocumentShares(c.Request.Context(), documentID)
	if err != nil {
		respondInternalError(c, "Failed to list shares", err)
		return
	}

	c.JSON(http.StatusOK, models.ShareListResponse{Shares: shares})
}

// DeleteShare removes a share
func (h *ShareHandler) DeleteShare(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "Invalid document ID format")
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		respondInvalidID(c, "Invalid user ID format")
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check ownership", err)
		return
	}
	if !isOwner {
		respondForbidden(c, "Only document owner can delete shares")
		return
	}

	err = h.store.DeleteDocumentShare(c.Request.Context(), documentID, targetUserID)
	if err != nil {
		respondInternalError(c, "Failed to delete share", err)
		return
	}

	c.Status(204)
}
// CreateShareLink generates a public share link
func (h *ShareHandler) CreateShareLink(c *gin.Context) {
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "Invalid document ID format")
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check ownership", err)
		return
	}
	if !isOwner {
		respondForbidden(c, "Only document owner can create share links")
		return
	}

	// Parse request body
	var req struct {
		Permission string `json:"permission" binding:"required,oneof=view edit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithValidationError(c, err)
		return
	}

	// Generate share token
	token, err := h.store.GenerateShareToken(c.Request.Context(), documentID, req.Permission)
	if err != nil {
		respondInternalError(c, "Failed to generate share link", err)
		return
	}

	// Get frontend URL from env
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	shareURL := fmt.Sprintf("%s/editor/%s?share=%s", frontendURL, documentID.String(), token)

	c.JSON(http.StatusOK, gin.H{
		"url":        shareURL,
		"token":      token,
		"permission": req.Permission,
	})
}

// GetShareLink retrieves the current public share link
func (h *ShareHandler) GetShareLink(c *gin.Context) {
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "Invalid document ID format")
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check ownership", err)
		return
	}
	if !isOwner {
		respondForbidden(c, "Only document owner can view share links")
		return
	}

	token, exists, err := h.store.GetShareToken(c.Request.Context(), documentID)
	if err != nil {
		respondInternalError(c, "Failed to get share link", err)
		return
	}
	if !exists {
		respondNotFound(c, "share link")
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	shareURL := fmt.Sprintf("%s/editor/%s?share=%s", frontendURL, documentID.String(), token)

	c.JSON(http.StatusOK, gin.H{
		"url":   shareURL,
		"token": token,
	})
}

// RevokeShareLink removes the public share link
func (h *ShareHandler) RevokeShareLink(c *gin.Context) {
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "Invalid document ID format")
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check ownership", err)
		return
	}
	if !isOwner {
		respondForbidden(c, "Only document owner can revoke share links")
		return
	}

	err = h.store.RevokeShareToken(c.Request.Context(), documentID)
	if err != nil {
		respondInternalError(c, "Failed to revoke share link", err)
		return
	}

	// c.JSON(http.StatusOK, gin.H{"message": "Share link revoked successfully"})
	c.Status(204)
}