package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Announcement system errors
var (
	ErrAnnouncementNotFound = infraerrors.NotFound("ANNOUNCEMENT_NOT_FOUND", "announcement not found")
)

// Announcement represents a system announcement
type Announcement struct {
	ID          int64
	Title       string
	Content     string
	ContentType string // markdown | html | url
	Priority    int
	Status      string // active | inactive
	PublishedAt *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AnnouncementRead represents a user's read record for an announcement
type AnnouncementRead struct {
	ID             int64
	UserID         int64
	AnnouncementID int64
	ReadAt         time.Time
}

// CreateAnnouncementInput is the input for creating an announcement
type CreateAnnouncementInput struct {
	Title       string
	Content     string
	ContentType string
	Priority    int
	Status      string
	PublishedAt *time.Time
	ExpiresAt   *time.Time
}

// UpdateAnnouncementInput is the input for updating an announcement
type UpdateAnnouncementInput struct {
	Title       *string
	Content     *string
	ContentType *string
	Priority    *int
	Status      *string
	PublishedAt *time.Time
	ExpiresAt   *time.Time
	// ClearPublishedAt and ClearExpiresAt are used to explicitly clear the time fields
	ClearPublishedAt bool
	ClearExpiresAt   bool
}

// AnnouncementRepository defines the announcement data access interface
type AnnouncementRepository interface {
	// Create creates a new announcement
	Create(ctx context.Context, input *CreateAnnouncementInput) (*Announcement, error)

	// Update updates an existing announcement
	Update(ctx context.Context, id int64, input *UpdateAnnouncementInput) (*Announcement, error)

	// Delete deletes an announcement
	Delete(ctx context.Context, id int64) error

	// GetByID returns an announcement by ID
	GetByID(ctx context.Context, id int64) (*Announcement, error)

	// List returns all announcements with pagination
	List(ctx context.Context, offset, limit int) ([]*Announcement, int, error)

	// GetActiveAnnouncements returns all active announcements that are published and not expired
	GetActiveAnnouncements(ctx context.Context) ([]*Announcement, error)

	// GetUnreadAnnouncements returns active announcements that the user hasn't read
	GetUnreadAnnouncements(ctx context.Context, userID int64) ([]*Announcement, error)

	// MarkAsRead marks an announcement as read for a user
	MarkAsRead(ctx context.Context, userID, announcementID int64) error

	// MarkAllAsRead marks all unread announcements as read for a user
	MarkAllAsRead(ctx context.Context, userID int64, announcementIDs []int64) error

	// IsRead checks if a user has read an announcement
	IsRead(ctx context.Context, userID, announcementID int64) (bool, error)
}
