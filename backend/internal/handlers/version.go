package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/M1ngdaXie/realtime-collab/internal/auth"
	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VersionHandler struct {
	store *store.PostgresStore
}

func NewVersionHandler(s *store.PostgresStore) *VersionHandler {
	return &VersionHandler{store: s}
}

// CreateVersion creates a manual snapshot (requires edit permission)
func (h *VersionHandler) CreateVersion(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "Invalid document ID")
		return
	}

	// Check edit permission (only editors can create versions)
	canEdit, err := h.store.CanEditDocument(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check permissions", err)
		return
	}
	if !canEdit {
		respondForbidden(c, "Edit permission required to create versions")
		return
	}

	// Parse multipart form data
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		respondBadRequest(c, "Invalid multipart form")
		return
	}

	// Get version label (optional)
	versionLabel := c.PostForm("version_label")
	var labelPtr *string
	if versionLabel != "" {
		labelPtr = &versionLabel
	}

	// Get text preview (required)
	textPreview := c.PostForm("text_preview")
	if textPreview == "" {
		respondBadRequest(c, "text_preview is required")
		return
	}

	// Get Yjs snapshot binary (required)
	file, _, err := c.Request.FormFile("yjs_snapshot")
	if err != nil {
		respondBadRequest(c, "yjs_snapshot file is required")
		return
	}
	defer file.Close()

	snapshotData, err := io.ReadAll(file)
	if err != nil || len(snapshotData) == 0 {
		respondBadRequest(c, "Failed to read snapshot data")
		return
	}

	// Validate snapshot size (max 10MB)
	if len(snapshotData) > 10*1024*1024 {
		respondBadRequest(c, "Snapshot too large (max 10MB)")
		return
	}

	// Create version (manual snapshot)
	version, err := h.store.CreateDocumentVersion(
		c.Request.Context(),
		documentID,
		*userID,
		snapshotData,
		&textPreview,
		labelPtr,
		false, // is_auto_generated = false
	)
	if err != nil {
		respondInternalError(c, "Failed to create version", err)
		return
	}

	c.JSON(http.StatusCreated, version)
}

// ListVersions returns paginated version history (requires view permission)
func (h *VersionHandler) ListVersions(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "Invalid document ID")
		return
	}

	// Check view permission
	canView, err := h.store.CanViewDocument(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check permissions", err)
		return
	}
	if !canView {
		respondForbidden(c, "View permission required")
		return
	}

	// Parse pagination params
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100 // Max limit
	}

	versions, total, err := h.store.ListDocumentVersions(c.Request.Context(), documentID, limit, offset)
	if err != nil {
		respondInternalError(c, "Failed to list versions", err)
		return
	}

	c.JSON(http.StatusOK, models.VersionListResponse{
		Versions: versions,
		Total:    total,
	})
}

// GetVersionSnapshot returns the Yjs binary snapshot for a specific version
func (h *VersionHandler) GetVersionSnapshot(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	versionID, err := uuid.Parse(c.Param("versionId"))
	if err != nil {
		respondBadRequest(c, "Invalid version ID")
		return
	}

	// Get version
	version, err := h.store.GetDocumentVersion(c.Request.Context(), versionID)
	if err != nil {
		respondNotFound(c, "version")
		return
	}

	// Check permission on parent document
	canView, err := h.store.CanViewDocument(c.Request.Context(), version.DocumentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check permissions", err)
		return
	}
	if !canView {
		respondForbidden(c, "View permission required")
		return
	}

	// Return binary snapshot
	c.Data(http.StatusOK, "application/octet-stream", version.YjsSnapshot)
}

// RestoreVersion creates a new version from an old snapshot (non-destructive)
func (h *VersionHandler) RestoreVersion(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		respondUnauthorized(c, "Authentication required")
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "Invalid document ID")
		return
	}

	// Check edit permission
	canEdit, err := h.store.CanEditDocument(c.Request.Context(), documentID, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check permissions", err)
		return
	}
	if !canEdit {
		respondForbidden(c, "Edit permission required to restore versions")
		return
	}

	var req models.RestoreVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithValidationError(c, err)
		return
	}

	// Get the version to restore
	oldVersion, err := h.store.GetDocumentVersion(c.Request.Context(), req.VersionID)
	if err != nil {
		respondNotFound(c, "version")
		return
	}

	// Verify version belongs to this document
	if oldVersion.DocumentID != documentID {
		respondBadRequest(c, "Version does not belong to this document")
		return
	}

	// Update current document state with old snapshot
	if err := h.store.UpdateDocumentState(documentID, oldVersion.YjsSnapshot); err != nil {
		respondInternalError(c, "Failed to restore document state", err)
		return
	}

	// Create new version entry marking it as a restore
	restoreLabel := fmt.Sprintf("Restored from version %d", oldVersion.VersionNumber)
	newVersion, err := h.store.CreateDocumentVersion(
		c.Request.Context(),
		documentID,
		*userID,
		oldVersion.YjsSnapshot,
		oldVersion.TextPreview,
		&restoreLabel,
		false, // Manual restore
	)
	if err != nil {
		respondInternalError(c, "Failed to create restore version", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Version restored successfully",
		"new_version": newVersion,
	})
}
