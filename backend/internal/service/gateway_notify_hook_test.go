//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type balanceNotifyCall struct {
	user       *User
	oldBalance float64
	cost       float64
}

type quotaNotifyCall struct {
	account    *Account
	cost       float64
	quotaState *AccountQuotaState
}

type gatewayNotifySpy struct {
	balanceCalls chan balanceNotifyCall
	quotaCalls   chan quotaNotifyCall
}

func newGatewayNotifySpy() *gatewayNotifySpy {
	return &gatewayNotifySpy{
		balanceCalls: make(chan balanceNotifyCall, 1),
		quotaCalls:   make(chan quotaNotifyCall, 1),
	}
}

func (s *gatewayNotifySpy) CheckBalanceAfterDeduction(_ context.Context, user *User, oldBalance, cost float64) {
	s.balanceCalls <- balanceNotifyCall{
		user:       user,
		oldBalance: oldBalance,
		cost:       cost,
	}
}

func (s *gatewayNotifySpy) CheckAccountQuotaAfterIncrement(_ context.Context, account *Account, cost float64, quotaState *AccountQuotaState) {
	s.quotaCalls <- quotaNotifyCall{
		account:    account,
		cost:       cost,
		quotaState: quotaState,
	}
}

func TestFinalizePostUsageBilling_TriggersBalanceNotification(t *testing.T) {
	spy := newGatewayNotifySpy()
	params := &postUsageBillingParams{
		Cost: &CostBreakdown{
			ActualCost: 5,
		},
		User: &User{
			ID:      1,
			Balance: 20,
		},
		Account: &Account{
			ID: 10,
		},
	}
	deps := &billingDeps{
		billingCacheService: &BillingCacheService{},
		deferredService:     &DeferredService{},
		balanceNotifier:     spy,
	}

	finalizePostUsageBilling(params, deps)

	select {
	case call := <-spy.balanceCalls:
		require.Same(t, params.User, call.user)
		require.Equal(t, 20.0, call.oldBalance)
		require.Equal(t, 5.0, call.cost)
	case <-time.After(time.Second):
		t.Fatal("expected balance notification")
	}
}

func TestFinalizePostUsageBilling_TriggersAccountQuotaNotification(t *testing.T) {
	spy := newGatewayNotifySpy()
	params := &postUsageBillingParams{
		Cost: &CostBreakdown{
			TotalCost: 4,
		},
		Account: &Account{
			ID:       7,
			Name:     "quota-account",
			Platform: PlatformAnthropic,
			Type:     AccountTypeAPIKey,
		},
		AccountRateMultiplier: 2.5,
	}
	deps := &billingDeps{
		billingCacheService: &BillingCacheService{},
		deferredService:     &DeferredService{},
		balanceNotifier:     spy,
	}

	finalizePostUsageBilling(params, deps)

	select {
	case call := <-spy.quotaCalls:
		require.Same(t, params.Account, call.account)
		require.Equal(t, 10.0, call.cost)
		require.Nil(t, call.quotaState)
	case <-time.After(time.Second):
		t.Fatal("expected account quota notification")
	}
}

func TestNewGatewayService_NilBalanceNotifyServiceStaysNil(t *testing.T) {
	svc := NewGatewayService(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	require.True(t, svc.balanceNotifier == nil)
}
