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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ownership"})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner can share documents"})
		return
	}

	var req models.CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user by email
	targetUser, err := h.store.GetUserByEmail(c.Request.Context(), req.UserEmail)
	if err != nil || targetUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Create share
	share, err := h.store.CreateDocumentShare(
		c.Request.Context(),
		documentID,
		targetUser.ID,
		req.Permission,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create share"})
		return
	}

	c.JSON(http.StatusCreated, share)
}

// ListShares lists all shares for a document
func (h *ShareHandler) ListShares(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ownership"})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner can view shares"})
		return
	}

	shares, err := h.store.ListDocumentShares(c.Request.Context(), documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list shares"})
		return
	}

	c.JSON(http.StatusOK, models.ShareListResponse{Shares: shares})
}

// DeleteShare removes a share
func (h *ShareHandler) DeleteShare(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ownership"})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner can delete shares"})
		return
	}

	err = h.store.DeleteDocumentShare(c.Request.Context(), documentID, targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Share deleted successfully"})
}
// CreateShareLink generates a public share link
func (h *ShareHandler) CreateShareLink(c *gin.Context) {
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ownership"})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only document owner can create share links"})
		return
	}

	// Parse request body
	var req struct {
		Permission string `json:"permission" binding:"required,oneof=view edit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Permission must be 'view' or 'edit'"})
		return
	}

	// Generate share token
	token, err := h.store.GenerateShareToken(c.Request.Context(), documentID, req.Permission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate share link"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ownership"})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only document owner can view share links"})
		return
	}

	token, exists, err := h.store.GetShareToken(c.Request.Context(), documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get share link"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "No public share link exists"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if user is owner
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), documentID, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ownership"})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only document owner can revoke share links"})
		return
	}

	err = h.store.RevokeShareToken(c.Request.Context(), documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke share link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Share link revoked"})
}