//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type notifyUserRepoStub struct {
	user       *User
	updateCall int
}

func (s *notifyUserRepoStub) Create(context.Context, *User) error { return nil }
func (s *notifyUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	cloned := *s.user
	if s.user.BalanceNotifyThreshold != nil {
		value := *s.user.BalanceNotifyThreshold
		cloned.BalanceNotifyThreshold = &value
	}
	cloned.BalanceNotifyExtraEmails = append([]NotifyEmailEntry(nil), s.user.BalanceNotifyExtraEmails...)
	return &cloned, nil
}
func (s *notifyUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	return nil, ErrUserNotFound
}
func (s *notifyUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	return nil, ErrUserNotFound
}
func (s *notifyUserRepoStub) Update(_ context.Context, user *User) error {
	s.updateCall++
	cloned := *user
	if user.BalanceNotifyThreshold != nil {
		value := *user.BalanceNotifyThreshold
		cloned.BalanceNotifyThreshold = &value
	}
	cloned.BalanceNotifyExtraEmails = append([]NotifyEmailEntry(nil), user.BalanceNotifyExtraEmails...)
	s.user = &cloned
	return nil
}
func (s *notifyUserRepoStub) Delete(context.Context, int64) error { return nil }
func (s *notifyUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *notifyUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *notifyUserRepoStub) UpdateBalance(context.Context, int64, float64) error { return nil }
func (s *notifyUserRepoStub) DeductBalance(context.Context, int64, float64) error { return nil }
func (s *notifyUserRepoStub) UpdateConcurrency(context.Context, int64, int) error { return nil }
func (s *notifyUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }
func (s *notifyUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *notifyUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (s *notifyUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (s *notifyUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error { return nil }
func (s *notifyUserRepoStub) EnableTotp(context.Context, int64) error                { return nil }
func (s *notifyUserRepoStub) DisableTotp(context.Context, int64) error               { return nil }

type notifyEmailCacheStub struct {
	data      *VerificationCodeData
	deleteHit bool
	userRate  int64
}

func (s *notifyEmailCacheStub) GetVerificationCode(context.Context, string) (*VerificationCodeData, error) {
	return nil, nil
}
func (s *notifyEmailCacheStub) SetVerificationCode(context.Context, string, *VerificationCodeData, time.Duration) error {
	return nil
}
func (s *notifyEmailCacheStub) DeleteVerificationCode(context.Context, string) error { return nil }
func (s *notifyEmailCacheStub) GetNotifyVerifyCode(context.Context, string) (*VerificationCodeData, error) {
	return s.data, nil
}
func (s *notifyEmailCacheStub) SetNotifyVerifyCode(_ context.Context, _ string, data *VerificationCodeData, _ time.Duration) error {
	s.data = data
	return nil
}
func (s *notifyEmailCacheStub) DeleteNotifyVerifyCode(context.Context, string) error {
	s.deleteHit = true
	s.data = nil
	return nil
}
func (s *notifyEmailCacheStub) GetPasswordResetToken(context.Context, string) (*PasswordResetTokenData, error) {
	return nil, nil
}
func (s *notifyEmailCacheStub) SetPasswordResetToken(context.Context, string, *PasswordResetTokenData, time.Duration) error {
	return nil
}
func (s *notifyEmailCacheStub) DeletePasswordResetToken(context.Context, string) error { return nil }
func (s *notifyEmailCacheStub) IsPasswordResetEmailInCooldown(context.Context, string) bool {
	return false
}
func (s *notifyEmailCacheStub) SetPasswordResetEmailCooldown(context.Context, string, time.Duration) error {
	return nil
}
func (s *notifyEmailCacheStub) IncrNotifyCodeUserRate(context.Context, int64, time.Duration) (int64, error) {
	s.userRate++
	return s.userRate, nil
}
func (s *notifyEmailCacheStub) GetNotifyCodeUserRate(context.Context, int64) (int64, error) {
	return s.userRate, nil
}

func TestUpdateProfile_UpdatesBalanceNotifyFields(t *testing.T) {
	repo := &notifyUserRepoStub{
		user: &User{
			ID:                         1,
			Email:                      "user@example.com",
			Username:                   "old",
			Concurrency:                2,
			BalanceNotifyEnabled:       false,
			BalanceNotifyThresholdType: thresholdTypeFixed,
		},
	}
	svc := NewUserService(repo, nil, nil)
	threshold := 7.5
	enabled := true

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		Username:               notifyStringPtr("new"),
		BalanceNotifyEnabled:   &enabled,
		BalanceNotifyThreshold: &threshold,
	})
	require.NoError(t, err)
	require.Equal(t, "new", updated.Username)
	require.True(t, updated.BalanceNotifyEnabled)
	require.NotNil(t, updated.BalanceNotifyThreshold)
	require.Equal(t, 7.5, *updated.BalanceNotifyThreshold)
}

func TestVerifyAndAddNotifyEmail_AppendsVerifiedEntry(t *testing.T) {
	repo := &notifyUserRepoStub{
		user: &User{
			ID:                       1,
			BalanceNotifyExtraEmails: []NotifyEmailEntry{},
		},
	}
	cache := &notifyEmailCacheStub{
		data: &VerificationCodeData{
			Code:      "123456",
			Attempts:  0,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(verifyCodeTTL),
		},
	}
	svc := NewUserService(repo, nil, nil)

	err := svc.VerifyAndAddNotifyEmail(context.Background(), 1, "notify@example.com", "123456", cache)
	require.NoError(t, err)
	require.True(t, cache.deleteHit)
	require.Len(t, repo.user.BalanceNotifyExtraEmails, 1)
	require.Equal(t, "notify@example.com", repo.user.BalanceNotifyExtraEmails[0].Email)
	require.True(t, repo.user.BalanceNotifyExtraEmails[0].Verified)
	require.False(t, repo.user.BalanceNotifyExtraEmails[0].Disabled)
}

func TestRemoveNotifyEmail_RemovesCaseInsensitive(t *testing.T) {
	repo := &notifyUserRepoStub{
		user: &User{
			ID: 1,
			BalanceNotifyExtraEmails: []NotifyEmailEntry{
				{Email: "One@example.com", Verified: true},
				{Email: "Two@example.com", Verified: true},
			},
		},
	}
	svc := NewUserService(repo, nil, nil)

	err := svc.RemoveNotifyEmail(context.Background(), 1, "one@example.com")
	require.NoError(t, err)
	require.Len(t, repo.user.BalanceNotifyExtraEmails, 1)
	require.Equal(t, "Two@example.com", repo.user.BalanceNotifyExtraEmails[0].Email)
}

func TestToggleNotifyEmail_UpdatesDisabledState(t *testing.T) {
	repo := &notifyUserRepoStub{
		user: &User{
			ID: 1,
			BalanceNotifyExtraEmails: []NotifyEmailEntry{
				{Email: "notify@example.com", Verified: true, Disabled: false},
			},
		},
	}
	svc := NewUserService(repo, nil, nil)

	err := svc.ToggleNotifyEmail(context.Background(), 1, "notify@example.com", true)
	require.NoError(t, err)
	require.True(t, repo.user.BalanceNotifyExtraEmails[0].Disabled)
}

func TestVerifyAndAddNotifyEmail_TooManyEntries(t *testing.T) {
	repo := &notifyUserRepoStub{
		user: &User{
			ID: 1,
			BalanceNotifyExtraEmails: []NotifyEmailEntry{
				{Email: "a@example.com", Verified: true},
				{Email: "b@example.com", Verified: true},
				{Email: "c@example.com", Verified: true},
			},
		},
	}
	cache := &notifyEmailCacheStub{
		data: &VerificationCodeData{
			Code:      "123456",
			Attempts:  0,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(verifyCodeTTL),
		},
	}
	svc := NewUserService(repo, nil, nil)

	err := svc.VerifyAndAddNotifyEmail(context.Background(), 1, "d@example.com", "123456", cache)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
}

func notifyStringPtr(v string) *string {
	return &v
}
