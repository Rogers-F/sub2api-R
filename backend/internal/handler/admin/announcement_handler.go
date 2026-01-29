package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler handles admin announcement-related requests
type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
}

// NewAnnouncementHandler creates a new admin AnnouncementHandler
func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
	}
}

// AnnouncementResponse represents an announcement for admin API response
type AnnouncementResponse struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	ContentType string  `json:"content_type"`
	Priority    int     `json:"priority"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// CreateAnnouncementRequest represents the create announcement request
type CreateAnnouncementRequest struct {
	Title       string  `json:"title" binding:"required"`
	Content     string  `json:"content" binding:"required"`
	ContentType string  `json:"content_type"`
	Priority    int     `json:"priority"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
	ExpiresAt   *string `json:"expires_at"`
}

// UpdateAnnouncementRequest represents the update announcement request
type UpdateAnnouncementRequest struct {
	Title            *string `json:"title"`
	Content          *string `json:"content"`
	ContentType      *string `json:"content_type"`
	Priority         *int    `json:"priority"`
	Status           *string `json:"status"`
	PublishedAt      *string `json:"published_at"`
	ExpiresAt        *string `json:"expires_at"`
	ClearPublishedAt bool    `json:"clear_published_at"`
	ClearExpiresAt   bool    `json:"clear_expires_at"`
}

// parseFlexibleTime parses time strings in RFC3339 format with optional milliseconds
// This supports both "2006-01-02T15:04:05Z" and "2006-01-02T15:04:05.000Z" formats
func parseFlexibleTime(s string) (time.Time, error) {
	// Try RFC3339 first (without milliseconds)
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	// Try RFC3339Nano (with milliseconds/nanoseconds)
	t, err = time.Parse(time.RFC3339Nano, s)
	if err == nil {
		return t, nil
	}
	// Try ISO 8601 with milliseconds explicitly
	t, err = time.Parse("2006-01-02T15:04:05.000Z07:00", s)
	if err == nil {
		return t, nil
	}
	return time.Time{}, err
}

func toAnnouncementResponse(a *service.Announcement) AnnouncementResponse {
	resp := AnnouncementResponse{
		ID:          a.ID,
		Title:       a.Title,
		Content:     a.Content,
		ContentType: a.ContentType,
		Priority:    a.Priority,
		Status:      a.Status,
		CreatedAt:   a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   a.UpdatedAt.Format(time.RFC3339),
	}
	if a.PublishedAt != nil {
		t := a.PublishedAt.Format(time.RFC3339)
		resp.PublishedAt = &t
	}
	if a.ExpiresAt != nil {
		t := a.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &t
	}
	return resp
}

// List returns all announcements with pagination
// GET /api/v1/admin/announcements
func (h *AnnouncementHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	announcements, total, err := h.announcementService.List(c.Request.Context(), offset, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]AnnouncementResponse, 0, len(announcements))
	for _, a := range announcements {
		items = append(items, toAnnouncementResponse(a))
	}

	response.Success(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get returns an announcement by ID
// GET /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	announcement, err := h.announcementService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, toAnnouncementResponse(announcement))
}

// Create creates a new announcement
// POST /api/v1/admin/announcements
func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.CreateAnnouncementInput{
		Title:       req.Title,
		Content:     req.Content,
		ContentType: req.ContentType,
		Priority:    req.Priority,
		Status:      req.Status,
	}

	if req.PublishedAt != nil && *req.PublishedAt != "" {
		t, err := parseFlexibleTime(*req.PublishedAt)
		if err != nil {
			response.BadRequest(c, "Invalid published_at format, expected RFC3339")
			return
		}
		input.PublishedAt = &t
	}

	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := parseFlexibleTime(*req.ExpiresAt)
		if err != nil {
			response.BadRequest(c, "Invalid expires_at format, expected RFC3339")
			return
		}
		input.ExpiresAt = &t
	}

	announcement, err := h.announcementService.Create(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Created(c, toAnnouncementResponse(announcement))
}

// Update updates an existing announcement
// PUT /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	var req UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.UpdateAnnouncementInput{
		Title:            req.Title,
		Content:          req.Content,
		ContentType:      req.ContentType,
		Priority:         req.Priority,
		Status:           req.Status,
		ClearPublishedAt: req.ClearPublishedAt,
		ClearExpiresAt:   req.ClearExpiresAt,
	}

	if req.PublishedAt != nil && *req.PublishedAt != "" {
		t, err := parseFlexibleTime(*req.PublishedAt)
		if err != nil {
			response.BadRequest(c, "Invalid published_at format, expected RFC3339")
			return
		}
		input.PublishedAt = &t
	}

	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := parseFlexibleTime(*req.ExpiresAt)
		if err != nil {
			response.BadRequest(c, "Invalid expires_at format, expected RFC3339")
			return
		}
		input.ExpiresAt = &t
	}

	announcement, err := h.announcementService.Update(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, toAnnouncementResponse(announcement))
}

// Delete deletes an announcement
// DELETE /api/v1/admin/announcements/:id
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid announcement ID")
		return
	}

	if err := h.announcementService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Announcement deleted"})
}
