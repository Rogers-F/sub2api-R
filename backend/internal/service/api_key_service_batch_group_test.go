//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type batchGroupAPIKeyRepoStub struct {
	ownedIDs       []int64
	keysByID       map[int64]string
	affected       int64
	updatedIDs     []int64
	updatedGroupID int64
}

func (s *batchGroupAPIKeyRepoStub) Create(context.Context, *APIKey) error {
	panic("unexpected Create call")
}

func (s *batchGroupAPIKeyRepoStub) GetByID(context.Context, int64) (*APIKey, error) {
	panic("unexpected GetByID call")
}

func (s *batchGroupAPIKeyRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	panic("unexpected GetKeyAndOwnerID call")
}

func (s *batchGroupAPIKeyRepoStub) GetByKey(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKey call")
}

func (s *batchGroupAPIKeyRepoStub) GetByKeyForAuth(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKeyForAuth call")
}

func (s *batchGroupAPIKeyRepoStub) Update(context.Context, *APIKey) error {
	panic("unexpected Update call")
}

func (s *batchGroupAPIKeyRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *batchGroupAPIKeyRepoStub) ListByUserID(context.Context, int64, pagination.PaginationParams, APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserID call")
}

func (s *batchGroupAPIKeyRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	return append([]int64(nil), s.ownedIDs...), nil
}

func (s *batchGroupAPIKeyRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	panic("unexpected CountByUserID call")
}

func (s *batchGroupAPIKeyRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	panic("unexpected ExistsByKey call")
}

func (s *batchGroupAPIKeyRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}

func (s *batchGroupAPIKeyRepoStub) SearchAPIKeys(context.Context, int64, string, int) ([]APIKey, error) {
	panic("unexpected SearchAPIKeys call")
}

func (s *batchGroupAPIKeyRepoStub) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected ClearGroupIDByGroupID call")
}

func (s *batchGroupAPIKeyRepoStub) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	panic("unexpected UpdateGroupIDByUserAndGroup call")
}

func (s *batchGroupAPIKeyRepoStub) BatchUpdateGroupIDByUserAndIDs(_ context.Context, _ int64, ids []int64, groupID int64) (int64, error) {
	s.updatedIDs = append([]int64(nil), ids...)
	s.updatedGroupID = groupID
	return s.affected, nil
}

func (s *batchGroupAPIKeyRepoStub) CountByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected CountByGroupID call")
}

func (s *batchGroupAPIKeyRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByUserID call")
}

func (s *batchGroupAPIKeyRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByGroupID call")
}

func (s *batchGroupAPIKeyRepoStub) ListKeysByUserAndIDs(_ context.Context, _ int64, ids []int64) ([]string, error) {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		if key, ok := s.keysByID[id]; ok {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *batchGroupAPIKeyRepoStub) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *batchGroupAPIKeyRepoStub) UpdateLastUsed(context.Context, int64, time.Time) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *batchGroupAPIKeyRepoStub) IncrementRateLimitUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementRateLimitUsage call")
}

func (s *batchGroupAPIKeyRepoStub) ResetRateLimitWindows(context.Context, int64) error {
	panic("unexpected ResetRateLimitWindows call")
}

func (s *batchGroupAPIKeyRepoStub) GetRateLimitData(context.Context, int64) (*APIKeyRateLimitData, error) {
	panic("unexpected GetRateLimitData call")
}

type batchGroupUserRepoStub struct {
	user *User
}

func (s *batchGroupUserRepoStub) Create(context.Context, *User) error {
	panic("unexpected Create call")
}
func (s *batchGroupUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	clone := *s.user
	return &clone, nil
}
func (s *batchGroupUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected GetByEmail call")
}
func (s *batchGroupUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}
func (s *batchGroupUserRepoStub) Update(context.Context, *User) error {
	panic("unexpected Update call")
}
func (s *batchGroupUserRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}
func (s *batchGroupUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *batchGroupUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (s *batchGroupUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}
func (s *batchGroupUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}
func (s *batchGroupUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}
func (s *batchGroupUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}
func (s *batchGroupUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}
func (s *batchGroupUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}
func (s *batchGroupUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}
func (s *batchGroupUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}
func (s *batchGroupUserRepoStub) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}
func (s *batchGroupUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}

type batchGroupGroupRepoStub struct {
	group *Group
}

func (s *batchGroupGroupRepoStub) Create(context.Context, *Group) error {
	panic("unexpected Create call")
}
func (s *batchGroupGroupRepoStub) GetByID(context.Context, int64) (*Group, error) {
	clone := *s.group
	return &clone, nil
}
func (s *batchGroupGroupRepoStub) GetByIDLite(context.Context, int64) (*Group, error) {
	panic("unexpected GetByIDLite call")
}
func (s *batchGroupGroupRepoStub) Update(context.Context, *Group) error {
	panic("unexpected Update call")
}
func (s *batchGroupGroupRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}
func (s *batchGroupGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade call")
}
func (s *batchGroupGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *batchGroupGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (s *batchGroupGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	panic("unexpected ListActive call")
}
func (s *batchGroupGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	panic("unexpected ListActiveByPlatform call")
}
func (s *batchGroupGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected ExistsByName call")
}
func (s *batchGroupGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected GetAccountCount call")
}
func (s *batchGroupGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected DeleteAccountGroupsByGroupID call")
}
func (s *batchGroupGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected GetAccountIDsByGroupIDs call")
}
func (s *batchGroupGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected BindAccountsToGroup call")
}
func (s *batchGroupGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders call")
}

type batchGroupAPIKeyCacheStub struct {
	deletedAuthKeys []string
}

func (s *batchGroupAPIKeyCacheStub) GetCreateAttemptCount(context.Context, int64) (int, error) {
	return 0, nil
}
func (s *batchGroupAPIKeyCacheStub) IncrementCreateAttemptCount(context.Context, int64) error {
	return nil
}
func (s *batchGroupAPIKeyCacheStub) DeleteCreateAttemptCount(context.Context, int64) error {
	return nil
}
func (s *batchGroupAPIKeyCacheStub) IncrementDailyUsage(context.Context, string) error {
	return nil
}
func (s *batchGroupAPIKeyCacheStub) SetDailyUsageExpiry(context.Context, string, time.Duration) error {
	return nil
}
func (s *batchGroupAPIKeyCacheStub) GetAuthCache(context.Context, string) (*APIKeyAuthCacheEntry, error) {
	return nil, nil
}
func (s *batchGroupAPIKeyCacheStub) SetAuthCache(context.Context, string, *APIKeyAuthCacheEntry, time.Duration) error {
	return nil
}
func (s *batchGroupAPIKeyCacheStub) DeleteAuthCache(_ context.Context, key string) error {
	s.deletedAuthKeys = append(s.deletedAuthKeys, key)
	return nil
}
func (s *batchGroupAPIKeyCacheStub) PublishAuthCacheInvalidation(context.Context, string) error {
	return nil
}
func (s *batchGroupAPIKeyCacheStub) SubscribeAuthCacheInvalidation(context.Context, func(string)) error {
	return nil
}

func TestAPIKeyService_BatchUpdateGroup_Success(t *testing.T) {
	repo := &batchGroupAPIKeyRepoStub{
		ownedIDs: []int64{1, 2},
		keysByID: map[int64]string{1: "sk-one", 2: "sk-two"},
		affected: 2,
	}
	userRepo := &batchGroupUserRepoStub{user: &User{ID: 7, AllowedGroups: []int64{10}}}
	groupRepo := &batchGroupGroupRepoStub{group: &Group{ID: 10, Status: StatusActive}}
	cache := &batchGroupAPIKeyCacheStub{}
	svc := &APIKeyService{
		apiKeyRepo:  repo,
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		userSubRepo: &userSubRepoNoop{},
		cache:       cache,
	}

	updated, err := svc.BatchUpdateGroup(context.Background(), 7, BatchUpdateAPIKeyGroupRequest{
		IDs:     []int64{1, 2, 2},
		GroupID: 10,
	})

	require.NoError(t, err)
	require.Equal(t, int64(2), updated)
	require.Equal(t, []int64{1, 2}, repo.updatedIDs)
	require.Equal(t, int64(10), repo.updatedGroupID)
	require.ElementsMatch(t, []string{svc.authCacheKey("sk-one"), svc.authCacheKey("sk-two")}, cache.deletedAuthKeys)
}

func TestAPIKeyService_BatchUpdateGroup_RejectsOwnershipMismatch(t *testing.T) {
	repo := &batchGroupAPIKeyRepoStub{ownedIDs: []int64{1}}
	svc := &APIKeyService{
		apiKeyRepo:  repo,
		userRepo:    &batchGroupUserRepoStub{user: &User{ID: 7, AllowedGroups: []int64{10}}},
		groupRepo:   &batchGroupGroupRepoStub{group: &Group{ID: 10, Status: StatusActive}},
		userSubRepo: &userSubRepoNoop{},
	}

	_, err := svc.BatchUpdateGroup(context.Background(), 7, BatchUpdateAPIKeyGroupRequest{
		IDs:     []int64{1, 2},
		GroupID: 10,
	})

	require.ErrorIs(t, err, ErrInsufficientPerms)
	require.Empty(t, repo.updatedIDs)
}

func TestAPIKeyService_BatchUpdateGroup_RejectsGroupNotAllowed(t *testing.T) {
	repo := &batchGroupAPIKeyRepoStub{ownedIDs: []int64{1, 2}}
	svc := &APIKeyService{
		apiKeyRepo:  repo,
		userRepo:    &batchGroupUserRepoStub{user: &User{ID: 7, AllowedGroups: []int64{9}}},
		groupRepo:   &batchGroupGroupRepoStub{group: &Group{ID: 10, Status: StatusActive, IsExclusive: true}},
		userSubRepo: &userSubRepoNoop{},
	}

	_, err := svc.BatchUpdateGroup(context.Background(), 7, BatchUpdateAPIKeyGroupRequest{
		IDs:     []int64{1, 2},
		GroupID: 10,
	})

	require.ErrorIs(t, err, ErrGroupNotAllowed)
	require.Empty(t, repo.updatedIDs)
}
