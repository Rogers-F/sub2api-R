package service

import (
	"context"
	"fmt"
)

// AnnouncementService handles announcement operations
type AnnouncementService struct {
	announcementRepo AnnouncementRepository
}

// NewAnnouncementService creates a new announcement service instance
func NewAnnouncementService(announcementRepo AnnouncementRepository) *AnnouncementService {
	return &AnnouncementService{
		announcementRepo: announcementRepo,
	}
}

// Create creates a new announcement
func (s *AnnouncementService) Create(ctx context.Context, input *CreateAnnouncementInput) (*Announcement, error) {
	// Set defaults
	if input.ContentType == "" {
		input.ContentType = AnnouncementContentTypeMarkdown
	}
	if input.Status == "" {
		input.Status = AnnouncementStatusActive
	}

	announcement, err := s.announcementRepo.Create(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}
	return announcement, nil
}

// Update updates an existing announcement
func (s *AnnouncementService) Update(ctx context.Context, id int64, input *UpdateAnnouncementInput) (*Announcement, error) {
	announcement, err := s.announcementRepo.Update(ctx, id, input)
	if err != nil {
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	return announcement, nil
}

// Delete deletes an announcement
func (s *AnnouncementService) Delete(ctx context.Context, id int64) error {
	if err := s.announcementRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

// GetByID returns an announcement by ID
func (s *AnnouncementService) GetByID(ctx context.Context, id int64) (*Announcement, error) {
	announcement, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get announcement: %w", err)
	}
	return announcement, nil
}

// List returns all announcements with pagination
func (s *AnnouncementService) List(ctx context.Context, offset, limit int) ([]*Announcement, int, error) {
	announcements, total, err := s.announcementRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list announcements: %w", err)
	}
	return announcements, total, nil
}

// GetUnreadAnnouncements returns unread active announcements for a user
func (s *AnnouncementService) GetUnreadAnnouncements(ctx context.Context, userID int64) ([]*Announcement, error) {
	announcements, err := s.announcementRepo.GetUnreadAnnouncements(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get unread announcements: %w", err)
	}
	return announcements, nil
}

// MarkAsRead marks an announcement as read for a user
func (s *AnnouncementService) MarkAsRead(ctx context.Context, userID, announcementID int64) error {
	if err := s.announcementRepo.MarkAsRead(ctx, userID, announcementID); err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}
	return nil
}

// MarkAllAsRead marks all provided announcements as read for a user
func (s *AnnouncementService) MarkAllAsRead(ctx context.Context, userID int64, announcementIDs []int64) error {
	if len(announcementIDs) == 0 {
		return nil
	}
	if err := s.announcementRepo.MarkAllAsRead(ctx, userID, announcementIDs); err != nil {
		return fmt.Errorf("mark all as read: %w", err)
	}
	return nil
}
