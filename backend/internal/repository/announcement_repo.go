package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type announcementRepository struct {
	sql *sql.DB
}

// NewAnnouncementRepository creates a new announcement repository
func NewAnnouncementRepository(_ *dbent.Client, sqlDB *sql.DB) service.AnnouncementRepository {
	return &announcementRepository{sql: sqlDB}
}

// Create creates a new announcement
func (r *announcementRepository) Create(ctx context.Context, input *service.CreateAnnouncementInput) (*service.Announcement, error) {
	query := `
		INSERT INTO announcements (title, content, content_type, priority, status, published_at, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	now := time.Now()
	ann := &service.Announcement{
		Title:       input.Title,
		Content:     input.Content,
		ContentType: input.ContentType,
		Priority:    input.Priority,
		Status:      input.Status,
		PublishedAt: input.PublishedAt,
		ExpiresAt:   input.ExpiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := r.sql.QueryRowContext(ctx, query,
		input.Title,
		input.Content,
		input.ContentType,
		input.Priority,
		input.Status,
		input.PublishedAt,
		input.ExpiresAt,
		now,
		now,
	).Scan(&ann.ID)
	if err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}
	return ann, nil
}

// Update updates an existing announcement
func (r *announcementRepository) Update(ctx context.Context, id int64, input *service.UpdateAnnouncementInput) (*service.Announcement, error) {
	// Build dynamic update query
	var updates []string
	var args []interface{}
	argIdx := 1

	if input.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *input.Title)
		argIdx++
	}
	if input.Content != nil {
		updates = append(updates, fmt.Sprintf("content = $%d", argIdx))
		args = append(args, *input.Content)
		argIdx++
	}
	if input.ContentType != nil {
		updates = append(updates, fmt.Sprintf("content_type = $%d", argIdx))
		args = append(args, *input.ContentType)
		argIdx++
	}
	if input.Priority != nil {
		updates = append(updates, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *input.Priority)
		argIdx++
	}
	if input.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *input.Status)
		argIdx++
	}
	if input.ClearPublishedAt {
		updates = append(updates, "published_at = NULL")
	} else if input.PublishedAt != nil {
		updates = append(updates, fmt.Sprintf("published_at = $%d", argIdx))
		args = append(args, *input.PublishedAt)
		argIdx++
	}
	if input.ClearExpiresAt {
		updates = append(updates, "expires_at = NULL")
	} else if input.ExpiresAt != nil {
		updates = append(updates, fmt.Sprintf("expires_at = $%d", argIdx))
		args = append(args, *input.ExpiresAt)
		argIdx++
	}

	// Always update updated_at
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	// Add id to args
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE announcements SET %s
		WHERE id = $%d
	`, joinStrings(updates, ", "), argIdx)

	result, err := r.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, service.ErrAnnouncementNotFound
	}

	return r.GetByID(ctx, id)
}

// Delete deletes an announcement
func (r *announcementRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM announcements WHERE id = $1`
	result, err := r.sql.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return service.ErrAnnouncementNotFound
	}
	return nil
}

// GetByID returns an announcement by ID
func (r *announcementRepository) GetByID(ctx context.Context, id int64) (*service.Announcement, error) {
	query := `
		SELECT id, title, content, content_type, priority, status, published_at, expires_at, created_at, updated_at
		FROM announcements
		WHERE id = $1
	`
	ann := &service.Announcement{}
	err := r.sql.QueryRowContext(ctx, query, id).Scan(
		&ann.ID,
		&ann.Title,
		&ann.Content,
		&ann.ContentType,
		&ann.Priority,
		&ann.Status,
		&ann.PublishedAt,
		&ann.ExpiresAt,
		&ann.CreatedAt,
		&ann.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrAnnouncementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get announcement by id: %w", err)
	}
	return ann, nil
}

// List returns all announcements with pagination
func (r *announcementRepository) List(ctx context.Context, offset, limit int) ([]*service.Announcement, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM announcements`
	if err := r.sql.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count announcements: %w", err)
	}

	// Get announcements
	query := `
		SELECT id, title, content, content_type, priority, status, published_at, expires_at, created_at, updated_at
		FROM announcements
		ORDER BY priority DESC, created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.sql.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list announcements: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var announcements []*service.Announcement
	for rows.Next() {
		ann := &service.Announcement{}
		if err := rows.Scan(
			&ann.ID,
			&ann.Title,
			&ann.Content,
			&ann.ContentType,
			&ann.Priority,
			&ann.Status,
			&ann.PublishedAt,
			&ann.ExpiresAt,
			&ann.CreatedAt,
			&ann.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan announcement: %w", err)
		}
		announcements = append(announcements, ann)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return announcements, total, nil
}

// GetActiveAnnouncements returns all active announcements that are published and not expired
func (r *announcementRepository) GetActiveAnnouncements(ctx context.Context) ([]*service.Announcement, error) {
	query := `
		SELECT id, title, content, content_type, priority, status, published_at, expires_at, created_at, updated_at
		FROM announcements
		WHERE status = 'active'
			AND (published_at IS NULL OR published_at <= NOW())
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY priority DESC, published_at DESC
	`
	rows, err := r.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get active announcements: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var announcements []*service.Announcement
	for rows.Next() {
		ann := &service.Announcement{}
		if err := rows.Scan(
			&ann.ID,
			&ann.Title,
			&ann.Content,
			&ann.ContentType,
			&ann.Priority,
			&ann.Status,
			&ann.PublishedAt,
			&ann.ExpiresAt,
			&ann.CreatedAt,
			&ann.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan announcement: %w", err)
		}
		announcements = append(announcements, ann)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return announcements, nil
}

// GetUnreadAnnouncements returns active announcements that the user hasn't read
func (r *announcementRepository) GetUnreadAnnouncements(ctx context.Context, userID int64) ([]*service.Announcement, error) {
	query := `
		SELECT a.id, a.title, a.content, a.content_type, a.priority, a.status, a.published_at, a.expires_at, a.created_at, a.updated_at
		FROM announcements a
		WHERE a.status = 'active'
			AND (a.published_at IS NULL OR a.published_at <= NOW())
			AND (a.expires_at IS NULL OR a.expires_at > NOW())
			AND NOT EXISTS (
				SELECT 1 FROM announcement_reads r
				WHERE r.announcement_id = a.id AND r.user_id = $1
			)
		ORDER BY a.priority DESC, a.published_at DESC
	`
	rows, err := r.sql.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get unread announcements: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var announcements []*service.Announcement
	for rows.Next() {
		ann := &service.Announcement{}
		if err := rows.Scan(
			&ann.ID,
			&ann.Title,
			&ann.Content,
			&ann.ContentType,
			&ann.Priority,
			&ann.Status,
			&ann.PublishedAt,
			&ann.ExpiresAt,
			&ann.CreatedAt,
			&ann.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan announcement: %w", err)
		}
		announcements = append(announcements, ann)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return announcements, nil
}

// MarkAsRead marks an announcement as read for a user
func (r *announcementRepository) MarkAsRead(ctx context.Context, userID, announcementID int64) error {
	query := `
		INSERT INTO announcement_reads (user_id, announcement_id, read_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, announcement_id) DO NOTHING
	`
	_, err := r.sql.ExecContext(ctx, query, userID, announcementID)
	if err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}
	return nil
}

// MarkAllAsRead marks all provided announcements as read for a user
func (r *announcementRepository) MarkAllAsRead(ctx context.Context, userID int64, announcementIDs []int64) error {
	if len(announcementIDs) == 0 {
		return nil
	}

	// Build bulk insert query
	query := `
		INSERT INTO announcement_reads (user_id, announcement_id, read_at)
		VALUES
	`
	var args []interface{}
	argIdx := 1
	for i, annID := range announcementIDs {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d, NOW())", argIdx, argIdx+1)
		args = append(args, userID, annID)
		argIdx += 2
	}
	query += " ON CONFLICT (user_id, announcement_id) DO NOTHING"

	_, err := r.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark all as read: %w", err)
	}
	return nil
}

// IsRead checks if a user has read an announcement
func (r *announcementRepository) IsRead(ctx context.Context, userID, announcementID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM announcement_reads WHERE user_id = $1 AND announcement_id = $2)`
	var exists bool
	if err := r.sql.QueryRowContext(ctx, query, userID, announcementID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check is read: %w", err)
	}
	return exists, nil
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
