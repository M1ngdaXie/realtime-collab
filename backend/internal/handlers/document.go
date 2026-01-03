package handlers

import (
	"fmt"
	"net/http"

	"github.com/M1ngdaXie/realtime-collab/internal/auth"
	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

  type DocumentHandler struct {
  	store *store.PostgresStore
  }

  func NewDocumentHandler(s *store.PostgresStore) *DocumentHandler {
  	return &DocumentHandler{store: s}
  }


  // CreateDocument creates a new document (requires auth)
func (h *DocumentHandler) CreateDocument(c *gin.Context) {
	fmt.Println("getting userId right now.... ")
	userID := auth.GetUserFromContext(c)
	fmt.Println(userID)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create document with owner_id
	doc, err := h.store.CreateDocumentWithOwner(req.Name, req.Type, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create document: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

  func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	userID := auth.GetUserFromContext(c)

	var docs []models.Document
	var err error

	if userID != nil {
		// Authenticated: show owned + shared documents
		docs, err = h.store.ListUserDocuments(c.Request.Context(), *userID)
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("we dont know you: %v", err)})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list documents"})
		return
	}

	c.JSON(http.StatusOK, models.DocumentListResponse{
		Documents: docs,
		Total:     len(docs),
	})
}


  func (h *DocumentHandler) GetDocument(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	userID := auth.GetUserFromContext(c)

	// Check permission if authenticated
	if userID != nil {
		canView, err := h.store.CanViewDocument(c.Request.Context(), id, *userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
			return
		}
		if !canView {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}else{
		c.JSON("this file is not public")
		return
	}

	doc, err := h.store.GetDocument(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, doc)
}
  // GetDocumentState returns the Yjs state for a document
  // GetDocumentState retrieves document state (requires view permission)
func (h *DocumentHandler) GetDocumentState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	userID := auth.GetUserFromContext(c)

	// Check permission if authenticated
	if userID != nil {
		canView, err := h.store.CanViewDocument(c.Request.Context(), id, *userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
			return
		}
		if !canView {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	doc, err := h.store.GetDocument(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", doc.YjsState)
}

  // UpdateDocumentState updates document state (requires edit permission)
func (h *DocumentHandler) UpdateDocumentState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check edit permission
	canEdit, err := h.store.CanEditDocument(c.Request.Context(), id, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
		return
	}
	if !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "Edit access denied"})
		return
	}

	var req models.UpdateStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateDocumentState(id, req.State); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update state"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "State updated successfully"})
}

  // DeleteDocument deletes a document (owner only)
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check ownership
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), id, *userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ownership"})
		return
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner can delete documents"})
		return
	}

	if err := h.store.DeleteDocument(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}