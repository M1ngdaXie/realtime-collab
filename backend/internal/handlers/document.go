package handlers

import (
	"net/http"

	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

  type DocumentHandler struct {
  	store *store.Store
  }

  func NewDocumentHandler(s *store.Store) *DocumentHandler {
  	return &DocumentHandler{store: s}
  }

  // CreateDocument creates a new document
  func (h *DocumentHandler) CreateDocument(c *gin.Context) {
  	var req models.CreateDocumentRequest
  	if err := c.ShouldBindJSON(&req); err != nil {
  		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
  		return
  	}

  	// Validate document type
  	if req.Type != models.DocumentTypeEditor && req.Type != models.DocumentTypeKanban {
  		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document type"})
  		return
  	}

  	doc, err := h.store.CreateDocument(req.Name, req.Type)
  	if err != nil {
  		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
  		return
  	}

  	c.JSON(http.StatusCreated, doc)
  }

  // ListDocuments returns all documents
  func (h *DocumentHandler) ListDocuments(c *gin.Context) {
  	documents, err := h.store.ListDocuments()
  	if err != nil {
  		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
  		return
  	}

  	if documents == nil {
  		documents = []models.Document{}
  	}

  	c.JSON(http.StatusOK, models.DocumentListResponse{
  		Documents: documents,
  		Total:     len(documents),
  	})
  }

  // GetDocument returns a single document
  func (h *DocumentHandler) GetDocument(c *gin.Context) {
  	idStr := c.Param("id")
  	id, err := uuid.Parse(idStr)
  	if err != nil {
  		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
  		return
  	}

  	doc, err := h.store.GetDocument(id)
  	if err != nil {
  		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
  		return
  	}

  	c.JSON(http.StatusOK, doc)
  }

  // GetDocumentState returns the Yjs state for a document
  func (h *DocumentHandler) GetDocumentState(c *gin.Context) {
  	idStr := c.Param("id")
  	id, err := uuid.Parse(idStr)
  	if err != nil {
  		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
  		return
  	}

  	doc, err := h.store.GetDocument(id)
  	if err != nil {
  		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
  		return
  	}

  	// Return binary state
  	if doc.YjsState == nil {
  		c.Data(http.StatusOK, "application/octet-stream", []byte{})
  		return
  	}

  	c.Data(http.StatusOK, "application/octet-stream", doc.YjsState)
  }

  // UpdateDocumentState updates the Yjs state for a document
  func (h *DocumentHandler) UpdateDocumentState(c *gin.Context) {
  	idStr := c.Param("id")
  	id, err := uuid.Parse(idStr)
  	if err != nil {
  		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
  		return
  	}

  	// Read binary body
  	state, err := c.GetRawData()
  	if err != nil {
  		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
  		return
  	}

  	err = h.store.UpdateDocumentState(id, state)
  	if err != nil {
  		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
  		return
  	}

  	c.JSON(http.StatusOK, gin.H{"message": "state updated successfully"})
  }

  // DeleteDocument deletes a document
  func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
  	idStr := c.Param("id")
  	id, err := uuid.Parse(idStr)
  	if err != nil {
  		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
  		return
  	}

  	err = h.store.DeleteDocument(id)
  	if err != nil {
  		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
  		return
  	}

  	c.JSON(http.StatusOK, gin.H{"message": "document deleted successfully"})
  }
