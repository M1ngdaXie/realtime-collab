package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/M1ngdaXie/realtime-collab/internal/auth"
	"github.com/M1ngdaXie/realtime-collab/internal/messagebus"
	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DocumentHandler struct {
	store      *store.PostgresStore
	messageBus messagebus.MessageBus
	serverID   string
	logger     *zap.Logger
}

func NewDocumentHandler(s *store.PostgresStore, msgBus messagebus.MessageBus, serverID string, logger *zap.Logger) *DocumentHandler {
	return &DocumentHandler{
		store:      s,
		messageBus: msgBus,
		serverID:   serverID,
		logger:     logger,
	}
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
	shareToken := c.Query("share")

	doc, err := h.store.GetDocument(id)
	if err != nil {
		respondNotFound(c, "document")
		return
	}

	// Check permission if authenticated
	if userID != nil {
		canView, err := h.store.CanViewDocument(c.Request.Context(), id, *userID)
		if err != nil {
			respondInternalError(c, "Failed to check permissions", err)
			return
		}
		if !canView && shareToken != "" {
			// Logged-in user without personal permission: fall back to share link
			valid, err := h.store.ValidateShareToken(c.Request.Context(), id, shareToken)
			if err != nil {
				respondInternalError(c, "Failed to validate share token", err)
				return
			}
			canView = valid
		}
		if !canView {
			respondForbidden(c, "Access denied")
			return
		}
	} else {
		// Unauthenticated: require valid share token or public doc
		if shareToken != "" {
			valid, err := h.store.ValidateShareToken(c.Request.Context(), id, shareToken)
			if err != nil {
				respondInternalError(c, "Failed to validate share token", err)
				return
			}
			if !valid {
				respondForbidden(c, "Invalid or expired share token")
				return
			}
		} else if !doc.Is_Public {
			respondForbidden(c, "This document is not public. Please sign in to access.")
			return
		}
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

	// Check edit permission (personal share OR edit-level share link)
	shareToken := c.Query("share")
	canEdit, err := h.store.CanEditDocument(c.Request.Context(), id, *userID)
	if err != nil {
		respondInternalError(c, "Failed to check permissions", err)
		return
	}
	if !canEdit && shareToken != "" {
		valid, err := h.store.ValidateShareToken(c.Request.Context(), id, shareToken)
		if err != nil {
			respondInternalError(c, "Failed to validate share token", err)
			return
		}
		if valid {
			perm, err := h.store.GetShareLinkPermission(c.Request.Context(), id)
			if err != nil {
				respondInternalError(c, "Failed to get token permission", err)
				return
			}
			canEdit = perm == "edit"
		}
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

	if streamID, seq, ok := h.addSnapshotMarker(c.Request.Context(), id); ok {
		if err := h.store.UpsertStreamCheckpoint(c.Request.Context(), id, streamID, seq); err != nil {
			if h.logger != nil {
				h.logger.Warn("Failed to upsert stream checkpoint after snapshot",
					zap.String("document_id", id.String()),
					zap.Error(err))
			}
		}
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

func (h *DocumentHandler) addSnapshotMarker(ctx context.Context, documentID uuid.UUID) (string, int64, bool) {
	if h.messageBus == nil {
		return "", 0, false
	}

	streamKey := "stream:" + documentID.String()
	seqKey := "seq:" + documentID.String()

	seq, err := h.messageBus.Incr(ctx, seqKey)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("Failed to increment snapshot sequence",
				zap.String("document_id", documentID.String()),
				zap.Error(err))
		}
		return "", 0, false
	}

	values := map[string]interface{}{
		"type":      "snapshot",
		"server_id": h.serverID,
		"seq":       seq,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	streamID, err := h.messageBus.XAdd(ctx, streamKey, 10000, true, values)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("Failed to add snapshot marker to stream",
				zap.String("stream_key", streamKey),
				zap.Error(err))
		}
		return "", 0, false
	}
	return streamID, seq, true
}

// GetDocumentPermission returns the user's permission level for a document
func (h *DocumentHandler) GetDocumentPermission(c *gin.Context) {
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondBadRequest(c, "Invalid document ID format")
		return
	}

	// 1. 先检查文档是否存在 (Good practice)
	_, err = h.store.GetDocument(documentID)
	if err != nil {
		respondNotFound(c, "document")
		return
	}

	userID := auth.GetUserFromContext(c)
	shareToken := c.Query("share")

	// 定义两个临时变量，用来存两边的结果
	var userPerm string  // 存 document_shares 的结果
	var tokenPerm string // 存 share_token 的结果

	// ====================================================
	// 步骤 A: 检查个人权限 (Base Permission)
	// ====================================================
	if userID != nil {
		perm, err := h.store.GetUserPermission(c.Request.Context(), documentID, *userID)
		if err != nil {
			respondInternalError(c, "Failed to get user permission", err)
			return
		}
		userPerm = perm
		// ⚠️ 注意：如果 perm 是空，这里不报错！继续往下走！
	}

	// ====================================================
	// 步骤 B: 检查 Token 权限 (Upgrade Permission)
	// ====================================================
	if shareToken != "" {
		// 先验证 Token 是否有效
		valid, err := h.store.ValidateShareToken(c.Request.Context(), documentID, shareToken)
		if err != nil {
			respondInternalError(c, "Failed to validate token", err)
			return
		}

		// 只有 Token 有效才去取权限
		if valid {
			p, err := h.store.GetShareLinkPermission(c.Request.Context(), documentID)
			if err != nil {
				respondInternalError(c, "Failed to get token permission", err)
				return
			}
			tokenPerm = p
			// 处理数据库老数据的 fallback
			if tokenPerm == "" {
				tokenPerm = "view"
			}
		}
	}

	// ====================================================
	// 步骤 C: ⚡️ 权限合并与计算 (The Brain)
	// ====================================================

	finalPermission := ""
	role := "viewer" // 默认角色

	// 1. 如果是 Owner，无敌，直接返回
	if userPerm == "owner" {
		finalPermission = "edit"
		role = "owner"
		// 直接返回，不用看 Token 了
		c.JSON(http.StatusOK, models.PermissionResponse{
			Permission: finalPermission,
			Role:       role,
		})
		return
	}

	// 2. 比较 User 和 Token，取最大值
	// 逻辑：只要任意一边给了 "edit"，那就是 "edit"
	if userPerm == "edit" || tokenPerm == "edit" {
		finalPermission = "edit"
		role = "editor"
	} else if userPerm == "view" || tokenPerm == "view" {
		finalPermission = "view"
		role = "viewer"
	}

	// ====================================================
	// 步骤 D: 最终判决
	// ====================================================
	if finalPermission == "" {
		// 既没个人权限，Token 也不对（或者没 Token）
		if userID == nil {
			respondUnauthorized(c, "Authentication required") // 没登录且没Token
		} else {
			respondForbidden(c, "You don't have permission") // 登录了但没权限
		}
		return
	}

	c.JSON(http.StatusOK, models.PermissionResponse{
		Permission: finalPermission,
		Role:       role,
	})
}
