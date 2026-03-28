//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type anthropic429RepoRecorder struct {
	mockAccountRepoForGemini
	setRateLimitedCalls int
	lastResetAt         time.Time
	updateSessionCalls  int
	lastSessionStatus   string
}

func (r *anthropic429RepoRecorder) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	r.setRateLimitedCalls++
	r.lastResetAt = resetAt
	return nil
}

func (r *anthropic429RepoRecorder) UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error {
	r.updateSessionCalls++
	r.lastSessionStatus = status
	return nil
}

func TestRateLimitService_Handle429_AnthropicPerWindowStoresChosenWindowType(t *testing.T) {
	repo := &anthropic429RepoRecorder{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-utilization", "0.95")
	headers.Set("anthropic-ratelimit-unified-5h-reset", "1770998400")
	headers.Set("anthropic-ratelimit-unified-7d-utilization", "0.80")
	headers.Set("anthropic-ratelimit-unified-7d-reset", "1771549200")
	body := []byte(`{"type":"error","error":{"message":"weekly limit exceeded request_id=req_abc123"}}`)

	svc.handle429(context.Background(), account, headers, body)

	require.Equal(t, 1, repo.setRateLimitedCalls)
	require.Equal(t, 1, repo.updateSessionCalls)
	require.Equal(t, "rejected", repo.lastSessionStatus)
}

func TestRateLimitService_Handle429_AnthropicUnifiedResetStoresDiagnosticDetail(t *testing.T) {
	repo := &anthropic429RepoRecorder{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{ID: 2, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-reset", "1771549200")
	body := []byte(`{"type":"error","error":{"message":"weekly limit exceeded request_id=req_abc123"}}`)

	svc.handle429(context.Background(), account, headers, body)

	require.Equal(t, 1, repo.setRateLimitedCalls)
	require.Equal(t, 1, repo.updateSessionCalls)
	require.Equal(t, "rejected", repo.lastSessionStatus)
}
