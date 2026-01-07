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
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	var req models.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithValidationError(c, err)
		return
	}

	// Create document with owner_id
	doc, err := h.store.CreateDocumentWithOwner(req.Name, req.Type, userID)
	if err != nil {
		respondInternalError(c, "Failed to create document", err)
		return
	}

	c.JSON(http.StatusCreated, doc)
}

  func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	fmt.Println("Getting userId, which is : ")
	fmt.Println(userID)
	if userID == nil {
		respondUnauthorized(c, "Authentication required to list documents")
		return
	}

	// Authenticated: show owned + shared documents
	docs, err := h.store.ListUserDocuments(c.Request.Context(), *userID)
	if err != nil {
		respondInternalError(c, "Failed to list documents", err)
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
		respondBadRequest(c, "Invalid document ID format")
		return
	}

	// First, check if document exists (404 takes precedence over 403)
	doc, err := h.store.GetDocument(id)
	if err != nil {
		respondNotFound(c, "document")
		return
	}

	userID := auth.GetUserFromContext(c)

	// Check permission if authenticated
	if userID != nil {
		canView, err := h.store.CanViewDocument(c.Request.Context(), id, *userID)
		if err != nil {
			respondInternalError(c, "Failed to check permissions", err)
			return
		}
		if !canView {
			respondForbidden(c, "Access denied")
			return
		}
	} else {
		// Unauthenticated users can only access public documents
		if !doc.Is_Public {
			respondForbidden(c, "This document is not public. Please sign in to access.")
			return
		}
	}

	c.JSON(http.StatusOK, doc)
}
  // GetDocumentState returns the Yjs state for a document
  // GetDocumentState retrieves document state (requires view permission)
func (h *DocumentHandler) GetDocumentState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "Invalid document ID format")
		return
	}

	userID := auth.GetUserFromContext(c)

	// Check permission if authenticated
	if userID != nil {
		canView, err := h.store.CanViewDocument(c.Request.Context(), id, *userID)
		if err != nil {
			respondInternalError(c, "Failed to check permissions", err)
			return
		}
		if !canView {
			respondForbidden(c, "Access denied")
			return
		}
	}

	doc, err := h.store.GetDocument(id)
	if err != nil {
		respondNotFound(c, "document")
		return
	}

	// Return empty byte slice if state is nil (new document)
	state := doc.YjsState
	if state == nil {
		state = []byte{}
	}

	c.Data(http.StatusOK, "application/octet-stream", state)
}

  // UpdateDocumentState updates document state (requires edit permission)
func (h *DocumentHandler) UpdateDocumentState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "Invalid document ID format")
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	// First, check if document exists (404 takes precedence over 403)
	_, err = h.store.GetDocument(id)
	if err != nil {
		respondNotFound(c, "document")
		return
	}

	// Check edit permission
	canEdit, err := h.store.CanEditDocument(c.Request.Context(), id, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check permissions", err)
		return
	}
	if !canEdit {
		respondForbidden(c, "Edit access denied")
		return
	}

	// Read binary data directly from request body
	state, err := c.GetRawData()
	if err != nil {
		respondBadRequest(c, "Failed to read request body")
		return
	}

	if len(state) == 0 {
		respondBadRequest(c, "Empty state data")
		return
	}

	if err := h.store.UpdateDocumentState(id, state); err != nil {
		respondInternalError(c, "Failed to update state", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "State updated successfully"})
}

  // DeleteDocument deletes a document (owner only)
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "Invalid document ID format")
		return
	}

	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	// First, check if document exists (404 takes precedence over 403)
	_, err = h.store.GetDocument(id)
	if err != nil {
		respondNotFound(c, "document")
		return
	}

	// Check ownership
	isOwner, err := h.store.IsDocumentOwner(c.Request.Context(), id, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check ownership", err)
		return
	}
	if !isOwner {
		respondForbidden(c, "Only document owner can delete documents")
		return
	}

	if err := h.store.DeleteDocument(id); err != nil {
		respondInternalError(c, "Failed to delete document", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}