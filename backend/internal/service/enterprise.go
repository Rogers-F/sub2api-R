package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrEnterpriseNotFound  = infraerrors.NotFound("ENTERPRISE_NOT_FOUND", "enterprise not found")
	ErrEnterpriseNameTaken = infraerrors.Conflict("ENTERPRISE_NAME_TAKEN", "enterprise name already exists")
	ErrEnterpriseNotActive = infraerrors.BadRequest("ENTERPRISE_NOT_ACTIVE", "enterprise is not active")
	ErrInvalidEnterpriseID = infraerrors.BadRequest("INVALID_ENTERPRISE_ID", "enterprise_id must be positive")
)

const AccountListEnterpriseUnassigned int64 = -1

type Enterprise struct {
	ID           int64
	Name         string
	Notes        *string
	Status       string
	AccountCount int64
	RPM          int64
	ErrorRate5m  float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type EnterpriseListFilters struct {
	Search string
	Status string
}

type CreateEnterpriseInput struct {
	Name   string
	Notes  *string
	Status string
}

type UpdateEnterpriseInput struct {
	Name   string
	Notes  *string
	Status string
}

type EnterpriseRepository interface {
	Create(ctx context.Context, enterprise *Enterprise) error
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters EnterpriseListFilters) ([]Enterprise, *pagination.PaginationResult, error)
	GetByID(ctx context.Context, id int64) (*Enterprise, error)
	GetActiveByID(ctx context.Context, id int64) (*Enterprise, error)
	Update(ctx context.Context, enterprise *Enterprise) error
	Delete(ctx context.Context, id int64) error
	AssignAccounts(ctx context.Context, enterpriseID int64, accountIDs []int64) (int64, error)
	UnassignAccounts(ctx context.Context, enterpriseID int64, accountIDs []int64) (int64, error)
}

type AccountEnterpriseSupport interface {
	ListWithEnterpriseFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode string, enterpriseID int64) ([]Account, *pagination.PaginationResult, error)
	ValidateActiveEnterprise(ctx context.Context, enterpriseID int64) error
}

type EnterpriseService struct {
	repo        EnterpriseRepository
	accountRepo AccountRepository
}

func NewEnterpriseService(repo EnterpriseRepository, accountRepo AccountRepository) *EnterpriseService {
	return &EnterpriseService{repo: repo, accountRepo: accountRepo}
}

func (s *EnterpriseService) List(ctx context.Context, page, pageSize int, filters EnterpriseListFilters) ([]Enterprise, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	enterprises, result, err := s.repo.ListWithFilters(ctx, params, normalizeEnterpriseListFilters(filters))
	if err != nil {
		return nil, 0, err
	}
	return enterprises, result.Total, nil
}

func (s *EnterpriseService) Get(ctx context.Context, id int64) (*Enterprise, error) {
	if id <= 0 {
		return nil, ErrInvalidEnterpriseID
	}
	return s.repo.GetByID(ctx, id)
}

func (s *EnterpriseService) Create(ctx context.Context, input CreateEnterpriseInput) (*Enterprise, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("ENTERPRISE_NAME_REQUIRED", "enterprise name is required")
	}
	status := normalizeEnterpriseStatus(input.Status)
	if status == "" {
		return nil, infraerrors.BadRequest("INVALID_ENTERPRISE_STATUS", "enterprise status must be active or disabled")
	}
	enterprise := &Enterprise{
		Name:   name,
		Notes:  normalizeEnterpriseNotes(input.Notes),
		Status: status,
	}
	if err := s.repo.Create(ctx, enterprise); err != nil {
		return nil, err
	}
	return enterprise, nil
}

func (s *EnterpriseService) Update(ctx context.Context, id int64, input UpdateEnterpriseInput) (*Enterprise, error) {
	if id <= 0 {
		return nil, ErrInvalidEnterpriseID
	}
	enterprise, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		enterprise.Name = name
	}
	if input.Notes != nil {
		enterprise.Notes = normalizeEnterpriseNotes(input.Notes)
	}
	if input.Status != "" {
		status := normalizeEnterpriseStatus(input.Status)
		if status == "" {
			return nil, infraerrors.BadRequest("INVALID_ENTERPRISE_STATUS", "enterprise status must be active or disabled")
		}
		enterprise.Status = status
	}
	if err := s.repo.Update(ctx, enterprise); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *EnterpriseService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidEnterpriseID
	}
	return s.repo.Delete(ctx, id)
}

func (s *EnterpriseService) ListAccounts(ctx context.Context, enterpriseID int64, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, int64, error) {
	if enterpriseID <= 0 {
		return nil, 0, ErrInvalidEnterpriseID
	}
	if _, err := s.repo.GetByID(ctx, enterpriseID); err != nil {
		return nil, 0, err
	}
	lister, ok := s.accountRepo.(AccountEnterpriseSupport)
	if !ok {
		return nil, 0, infraerrors.InternalServer("ENTERPRISE_ACCOUNT_FILTER_UNAVAILABLE", "enterprise account filter is unavailable")
	}
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	accounts, result, err := lister.ListWithEnterpriseFilters(ctx, params, platform, accountType, status, search, groupID, privacyMode, enterpriseID)
	if err != nil {
		return nil, 0, err
	}
	return accounts, result.Total, nil
}

func (s *EnterpriseService) AssignAccounts(ctx context.Context, enterpriseID int64, accountIDs []int64) (int64, error) {
	if enterpriseID <= 0 {
		return 0, ErrInvalidEnterpriseID
	}
	if _, err := s.repo.GetActiveByID(ctx, enterpriseID); err != nil {
		return 0, err
	}
	return s.repo.AssignAccounts(ctx, enterpriseID, accountIDs)
}

func (s *EnterpriseService) UnassignAccounts(ctx context.Context, enterpriseID int64, accountIDs []int64) (int64, error) {
	if enterpriseID <= 0 {
		return 0, ErrInvalidEnterpriseID
	}
	if _, err := s.repo.GetByID(ctx, enterpriseID); err != nil {
		return 0, err
	}
	return s.repo.UnassignAccounts(ctx, enterpriseID, accountIDs)
}

func normalizeEnterpriseListFilters(filters EnterpriseListFilters) EnterpriseListFilters {
	filters.Search = strings.TrimSpace(filters.Search)
	if len(filters.Search) > 100 {
		filters.Search = filters.Search[:100]
	}
	filters.Status = strings.TrimSpace(filters.Status)
	return filters
}

func normalizeEnterpriseStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return StatusActive
	}
	switch status {
	case StatusActive, StatusDisabled:
		return status
	default:
		return ""
	}
}

func normalizeEnterpriseNotes(notes *string) *string {
	if notes == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*notes)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
