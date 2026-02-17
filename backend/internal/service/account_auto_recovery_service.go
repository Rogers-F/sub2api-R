package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	autoRecoveryMaxAttempts = 3
	autoRecoveryRetryDelay  = 30 * time.Second
)

// AccountAutoRecoveryService coordinates automatic recovery of accounts
// that enter error state. When triggered, it performs non-streaming
// connectivity tests with fixed-interval retry (up to 3 attempts, 30s apart).
type AccountAutoRecoveryService struct {
	testService *AccountTestService
	accountRepo AccountRepository
	timingWheel *TimingWheelService
	inProgress  sync.Map // map[int64]struct{} — dedup concurrent recovery for same account
}

// NewAccountAutoRecoveryService creates a new AccountAutoRecoveryService.
func NewAccountAutoRecoveryService(
	testService *AccountTestService,
	accountRepo AccountRepository,
	timingWheel *TimingWheelService,
) *AccountAutoRecoveryService {
	return &AccountAutoRecoveryService{
		testService: testService,
		accountRepo: accountRepo,
		timingWheel: timingWheel,
	}
}

// TriggerRecovery initiates the auto-recovery process for the given account.
// If a recovery is already in progress for this account, the call is a no-op.
func (s *AccountAutoRecoveryService) TriggerRecovery(accountID int64) {
	if _, loaded := s.inProgress.LoadOrStore(accountID, struct{}{}); loaded {
		return
	}
	go s.attemptRecovery(accountID, 1)
}

// Stop unbinds the SetError callback so no new recovery attempts are triggered.
// In-flight attempts will finish naturally (DB/TimingWheel access may fail
// gracefully after shutdown).
func (s *AccountAutoRecoveryService) Stop() {
	s.accountRepo.SetOnErrorCallback(nil)
	log.Println("[AutoRecovery] Stopped — callback unbound")
}

// attemptRecovery performs a single connectivity test and either recovers the
// account or schedules a retry via the timing wheel.
func (s *AccountAutoRecoveryService) attemptRecovery(accountID int64, attempt int) {
	// scheduled tracks whether we successfully handed off to the next attempt.
	// If false on exit, inProgress must be cleaned up.
	scheduled := false
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[AutoRecovery] Account %d: panic during attempt %d: %v", accountID, attempt, r)
		}
		if !scheduled {
			s.inProgress.Delete(accountID)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if account is still in error status (may have been manually recovered)
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		log.Printf("[AutoRecovery] Account %d not found, aborting: %v", accountID, err)
		return
	}
	if account.Status != StatusError {
		log.Printf("[AutoRecovery] Account %d no longer in error status, aborting", accountID)
		return
	}

	log.Printf("[AutoRecovery] Testing account %d (attempt %d/%d)", accountID, attempt, autoRecoveryMaxAttempts)

	testErr := s.testService.TestAccountConnectionQuiet(ctx, accountID)
	if testErr == nil {
		log.Printf("[AutoRecovery] Account %d recovered on attempt %d", accountID, attempt)
		return
	}

	log.Printf("[AutoRecovery] Account %d test failed (attempt %d/%d): %v", accountID, attempt, autoRecoveryMaxAttempts, testErr)

	if attempt >= autoRecoveryMaxAttempts {
		log.Printf("[AutoRecovery] Account %d: max attempts reached, keeping error status", accountID)
		return
	}

	// Schedule next attempt via timing wheel.
	// Note: if the TimingWheel is stopped (server shutdown), the scheduled task
	// won't fire and inProgress won't be cleaned. This is acceptable because
	// during shutdown, recovery is no longer needed.
	if s.timingWheel == nil {
		log.Printf("[AutoRecovery] Account %d: timing wheel unavailable, giving up", accountID)
		return
	}
	nextAttempt := attempt + 1
	taskName := fmt.Sprintf("auto_recovery_%d_%d", accountID, nextAttempt)
	s.timingWheel.Schedule(taskName, autoRecoveryRetryDelay, func() {
		s.attemptRecovery(accountID, nextAttempt)
	})
	scheduled = true
}
