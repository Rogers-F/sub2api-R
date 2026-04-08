//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type redeemUserSubRepoStub struct {
	userSubRepoNoop

	sub           *UserSubscription
	updatedStatus string
	updatedNotes  string
	extendedAt    time.Time
}

func (r *redeemUserSubRepoStub) GetByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.UserID != userID || r.sub.GroupID != groupID {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *redeemUserSubRepoStub) ExtendExpiry(_ context.Context, id int64, expiresAt time.Time) error {
	r.extendedAt = expiresAt
	if r.sub != nil && r.sub.ID == id {
		r.sub.ExpiresAt = expiresAt
	}
	return nil
}

func (r *redeemUserSubRepoStub) UpdateStatus(_ context.Context, id int64, status string) error {
	r.updatedStatus = status
	if r.sub != nil && r.sub.ID == id {
		r.sub.Status = status
	}
	return nil
}

func (r *redeemUserSubRepoStub) UpdateNotes(_ context.Context, id int64, notes string) error {
	r.updatedNotes = notes
	if r.sub != nil && r.sub.ID == id {
		r.sub.Notes = notes
	}
	return nil
}

func TestRedeemService_ReduceOrCancelSubscription_ReducesExpiry(t *testing.T) {
	originalExpiry := time.Now().AddDate(0, 0, 10).Round(time.Second)
	repo := &redeemUserSubRepoStub{
		sub: &UserSubscription{
			ID:        7,
			UserID:    11,
			GroupID:   22,
			ExpiresAt: originalExpiry,
			Notes:     "init",
		},
	}
	service := &RedeemService{
		subscriptionService: &SubscriptionService{userSubRepo: repo},
	}

	err := service.reduceOrCancelSubscription(context.Background(), 11, 22, 3, "CODE-REDUCE")
	require.NoError(t, err)
	require.Empty(t, repo.updatedStatus)
	require.WithinDuration(t, originalExpiry.AddDate(0, 0, -3), repo.extendedAt, time.Second)
	require.Contains(t, repo.updatedNotes, "CODE-REDUCE")
	require.Contains(t, repo.updatedNotes, "3 天")
}

func TestRedeemService_ReduceOrCancelSubscription_CancelsWhenReductionExceedsRemaining(t *testing.T) {
	repo := &redeemUserSubRepoStub{
		sub: &UserSubscription{
			ID:        8,
			UserID:    12,
			GroupID:   23,
			ExpiresAt: time.Now().Add(36 * time.Hour),
			Notes:     "init",
		},
	}
	service := &RedeemService{
		subscriptionService: &SubscriptionService{userSubRepo: repo},
	}

	before := time.Now()
	err := service.reduceOrCancelSubscription(context.Background(), 12, 23, 5, "CODE-CANCEL")
	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusExpired, repo.updatedStatus)
	require.WithinDuration(t, time.Now(), repo.extendedAt, 2*time.Second)
	require.False(t, repo.extendedAt.Before(before))
	require.Contains(t, repo.updatedNotes, "CODE-CANCEL")
	require.Contains(t, repo.updatedNotes, "5 天")
}
