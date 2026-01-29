package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler handles announcement-related requests for users
type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
}

// NewAnnouncementHandler creates a new AnnouncementHandler
func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
	}
}

// AnnouncementResponse represents an announcement for API response
type AnnouncementResponse struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	ContentType string  `json:"content_type"`
	Priority    int     `json:"priority"`
	PublishedAt *string `json:"published_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// GetUnreadAnnouncements returns unread announcements for the current user
// GET /api/v1/announcements/unread
func (h *AnnouncementHandler) GetUnreadAnnouncements(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	announcements, err := h.announcementService.GetUnreadAnnouncements(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]AnnouncementResponse, 0, len(announcements))
	for _, a := range announcements {
		item := AnnouncementResponse{
			ID:          a.ID,
			Title:       a.Title,
			Content:     a.Content,
			ContentType: a.ContentType,
			Priority:    a.Priority,
			CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if a.PublishedAt != nil {
			t := a.PublishedAt.Format("2006-01-02T15:04:05Z")
			item.PublishedAt = &t
		}
		items = append(items, item)
	}

	response.Success(c, items)
}

// MarkAsReadRequest represents the mark as read request
type MarkAsReadRequest struct {
	AnnouncementIDs []int64 `json:"announcement_ids"`
}

// MarkAsRead marks an announcement as read for the current user
// POST /api/v1/announcements/:id/read
func (h *AnnouncementHandler) MarkAsRead(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	if err := h.announcementService.MarkAsRead(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Marked as read"})
}

// MarkAllAsRead marks all provided announcements as read
// POST /api/v1/announcements/read-all
func (h *AnnouncementHandler) MarkAllAsRead(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req MarkAsReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.announcementService.MarkAllAsRead(c.Request.Context(), subject.UserID, req.AnnouncementIDs); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "All marked as read"})
}
