package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

// Claude OAuth request-path refresh tuning. Refresh is lazy (triggered on the
// request path, not a background tick): a request whose access token is within
// claudeOAuthRefreshSkew of expiry refreshes it before forwarding.
const (
	claudeOAuthRefreshSkew = 3 * time.Minute
	// claudeOAuthRefreshTime bounds the upstream token call.
	claudeOAuthRefreshTime = 30 * time.Second
	// claudeOAuthLockTTL MUST comfortably exceed the whole critical section
	// (upstream refresh ≤ claudeOAuthRefreshTime + DB CAS + sibling write-back + cache
	// rebuild). If the cross-instance lock expired mid-refresh, a peer could acquire it
	// and double-refresh, racing the refresh_token rotation. Guarded by a unit test that
	// asserts claudeOAuthLockTTL > claudeOAuthRefreshTime by a safe margin.
	claudeOAuthLockTTL     = 2 * time.Minute
	claudeOAuthLockWait    = 200 * time.Millisecond
	claudeOAuthLockRetries = 25 // ~5s ceiling waiting for a peer's refresh
)

func claudeOAuthLockKey(accountID string) string {
	return "oauth:refresh_lock:claude:" + accountID
}

// claudeOAuthKeyedMu serializes refreshes for one subscription within this process
// (keyed by account_uuid). It is the in-process layer; cross-instance serialization
// is the Redis SetNX lock. Refcounted so the map does not grow unbounded.
var claudeOAuthKeyedMu = newKeyedMutex()

// EnsureClaudeOAuthFresh returns a non-expired access token for the channel's OAuth
// credential, refreshing on the request path if the cached token is within skew of
// expiry. Concurrency-safe: refreshes for one subscription are serialized (in-process
// keyed mutex + cross-instance Redis lock), every decision re-reads the DB under the
// lock (double-checked), and the persist is an exact-key CAS so a loser never clobbers
// the winner's rotation. rawKey is the credential as the relay currently sees it
// (possibly stale); channelID/proxyURL identify the channel and its egress.
func EnsureClaudeOAuthFresh(ctx context.Context, channelID int, rawKey, proxyURL string) (string, error) {
	key, err := ParseClaudeOAuthKey(rawKey)
	if err != nil {
		return "", err
	}
	if !key.NeedsRefresh(claudeOAuthRefreshSkew) {
		return key.AccessToken, nil
	}
	accountID := key.LockAccountID()
	if accountID == "" {
		return "", errors.New("claude oauth: cannot derive lock identity (missing account_uuid and refresh_token)")
	}

	// In-process serialization for this subscription.
	unlock := claudeOAuthKeyedMu.Lock(accountID)
	defer unlock()

	// Double-check against the DB — another goroutine/instance may have just rotated it.
	if fresh, ok := reloadFreshClaudeToken(channelID); ok {
		return fresh, nil
	}

	// Cross-instance lock. When Redis is enabled it is the ONLY guard against a
	// concurrent upstream refresh from a peer instance — and an upstream refresh is NOT
	// idempotent: it rotates the refresh_token at Anthropic, a side-effect the DB CAS
	// cannot undo. So if we cannot acquire the lock, we FAIL CLOSED (return an error for
	// the client to retry) rather than risk a double refresh that bricks the credential.
	if common.RedisEnabled && common.RDB != nil {
		lockKey := claudeOAuthLockKey(accountID)
		owner, acquired := acquireClaudeRefreshLock(lockKey)
		if !acquired {
			// A peer instance holds the lock; poll the DB until it lands a fresh token.
			for i := 0; i < claudeOAuthLockRetries; i++ {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(claudeOAuthLockWait):
				}
				if fresh, ok := reloadFreshClaudeToken(channelID); ok {
					return fresh, nil
				}
				if owner, acquired = acquireClaudeRefreshLock(lockKey); acquired {
					break
				}
			}
		}
		if !acquired {
			// Peer never landed a fresh token within the wait budget. Fail closed — do
			// NOT refresh ourselves (that would race the refresh_token rotation).
			if fresh, ok := reloadFreshClaudeToken(channelID); ok {
				return fresh, nil
			}
			return "", errors.New("claude oauth: refresh in progress, please retry")
		}
		defer func() { _ = common.RedisReleaseLock(lockKey, owner) }()
		// Re-check now that we hold the cross-instance lock.
		if fresh, ok := reloadFreshClaudeToken(channelID); ok {
			return fresh, nil
		}
		return refreshClaudeOnce(ctx, channelID, proxyURL)
	}

	// No Redis: the in-process keyed mutex fully serializes refreshes within this process
	// (the supported single-instance topology). Multi-instance WITHOUT Redis has no
	// cross-instance lock and is unsupported for OAuth channels.
	return refreshClaudeOnce(ctx, channelID, proxyURL)
}

// ForceRefreshClaudeChannel unconditionally refreshes a channel's OAuth credential
// (admin "refresh now" action), still serialized via the account lock, and returns
// the freshly persisted (parsed) key. Callers MUST redact before returning to clients.
func ForceRefreshClaudeChannel(ctx context.Context, channelID int) (*ClaudeOAuthKey, error) {
	ch, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, errors.New("claude oauth: channel not found")
	}
	if ch.Type != constant.ChannelTypeAnthropic {
		return nil, errors.New("claude oauth: channel type is not Anthropic")
	}
	otherSettings := ch.GetOtherSettings()
	if !otherSettings.IsOAuthMode() {
		return nil, errors.New("claude oauth: channel is not in oauth auth_mode")
	}
	key, err := ParseClaudeOAuthKey(ch.Key)
	if err != nil {
		return nil, err
	}
	accountID := key.LockAccountID()
	if accountID == "" {
		return nil, errors.New("claude oauth: cannot derive lock identity")
	}

	unlock := claudeOAuthKeyedMu.Lock(accountID)
	defer unlock()

	// With Redis enabled the cross-instance lock is mandatory: a forced refresh must not
	// race a peer's refresh and rotate the refresh_token twice. Fail closed if we cannot
	// acquire it within the wait budget.
	if common.RedisEnabled && common.RDB != nil {
		lockKey := claudeOAuthLockKey(accountID)
		owner, acquired := acquireClaudeRefreshLock(lockKey)
		for i := 0; !acquired && i < claudeOAuthLockRetries; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(claudeOAuthLockWait):
			}
			owner, acquired = acquireClaudeRefreshLock(lockKey)
		}
		if !acquired {
			return nil, errors.New("claude oauth: refresh in progress, please retry")
		}
		defer func() { _ = common.RedisReleaseLock(lockKey, owner) }()
	}

	if _, err := refreshClaudeOnce(ctx, channelID, ch.GetSetting().Proxy); err != nil {
		return nil, err
	}
	// Re-read the persisted credential to return the authoritative state.
	updated, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, err
	}
	return ParseClaudeOAuthKey(updated.Key)
}

// reloadFreshClaudeToken re-reads the channel from the DB and returns its access
// token iff it is currently fresh (not within skew of expiry). Any error / staleness
// yields ok=false.
func reloadFreshClaudeToken(channelID int) (string, bool) {
	ch, err := model.GetChannelById(channelID, true)
	if err != nil || ch == nil {
		return "", false
	}
	key, err := ParseClaudeOAuthKey(ch.Key)
	if err != nil {
		return "", false
	}
	if key.NeedsRefresh(claudeOAuthRefreshSkew) {
		return "", false
	}
	return key.AccessToken, true
}

func acquireClaudeRefreshLock(lockKey string) (string, bool) {
	owner, err := createStateHex(16)
	if err != nil {
		return "", false
	}
	ok, err := common.RedisSetNX(lockKey, owner, claudeOAuthLockTTL)
	if err != nil || !ok {
		return "", false
	}
	return owner, true
}

// refreshClaudeOnce performs exactly one upstream token refresh for the channel and
// persists it: exact-key CAS on the refreshing channel, then a sibling write-back so
// every channel sharing this subscription (same account_uuid) carries the rotated
// token, then a cache rebuild. It does NOT take the account lock — the caller holds it
// (the CAS is the safety backstop either way). Returns the fresh access token.
func refreshClaudeOnce(ctx context.Context, channelID int, proxyURL string) (string, error) {
	ch, err := model.GetChannelById(channelID, true)
	if err != nil {
		return "", err
	}
	if ch == nil {
		return "", errors.New("claude oauth: channel not found")
	}
	if ch.Type != constant.ChannelTypeAnthropic {
		return "", errors.New("claude oauth: channel type is not Anthropic")
	}
	otherSettings := ch.GetOtherSettings()
	if !otherSettings.IsOAuthMode() {
		return "", errors.New("claude oauth: channel is not in oauth auth_mode")
	}

	oldKey := strings.TrimSpace(ch.Key)
	parsed, err := ParseClaudeOAuthKey(oldKey)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.RefreshToken) == "" {
		return "", errors.New("claude oauth: refresh_token is required to refresh credential")
	}

	proxy := strings.TrimSpace(proxyURL)
	if proxy == "" {
		proxy = ch.GetSetting().Proxy
	}

	refreshCtx, cancel := context.WithTimeout(ctx, claudeOAuthRefreshTime)
	defer cancel()
	res, err := RefreshClaudeOAuthTokenWithProxy(refreshCtx, parsed.RefreshToken, proxy)
	if err != nil {
		return "", err
	}

	version := time.Now().UnixMilli()
	lastRefresh := time.Now().Format(time.RFC3339)
	expiresAt := res.ExpiresAt.Unix()

	// Fill any identity fields newly learned from the response (without overwriting
	// existing values), then apply the rotated token fields (map-preserve).
	selfRaw, err := ensureClaudeIdentity(oldKey, res)
	if err != nil {
		return "", err
	}
	selfNew, err := MergeRefreshedTokens(selfRaw, res.AccessToken, res.RefreshToken, expiresAt, version, lastRefresh)
	if err != nil {
		return "", err
	}

	accountUUID := strings.TrimSpace(parsed.AccountUUID)
	if accountUUID == "" {
		accountUUID = strings.TrimSpace(res.AccountUUID)
	}

	swapped, err := model.CompareAndSwapChannelKey(channelID, ch.Key, selfNew)
	if err != nil {
		return "", err
	}
	persisted := res.AccessToken
	if !swapped {
		// The row changed out-of-band (we hold the account lock, so this is NOT a
		// concurrent refresh — it is an admin edit). Re-converge via bounded CAS-merge so
		// our rotation IS persisted; never return a token we did not write to the DB.
		token, ok := casApplyRotation(channelID, accountUUID, res, expiresAt, version, lastRefresh)
		if !ok {
			return "", errors.New("claude oauth: refresh could not be reconciled with a concurrent key change")
		}
		persisted = token
	}

	// Sibling write-back: every other OAuth channel sharing this subscription must carry
	// the same rotated token (they share one upstream refresh_token lineage).
	if accountUUID != "" {
		writeClaudeSiblings(channelID, accountUUID, res, expiresAt, version, lastRefresh)
	}

	// Rebuild the in-memory channel cache + proxy client cache so the rotated token takes
	// effect immediately on this instance.
	model.InitChannelCache()
	ResetProxyClientCache()

	return persisted, nil
}

// casApplyRotation persists a rotated token onto one channel via bounded exact-key CAS.
// It re-reads the authoritative row on each miss, refuses to downgrade (skips when the
// stored TokenVersion is already >= ours), and only touches a channel whose account_uuid
// matches (when accountUUID is non-empty). Returns the access token actually persisted in
// the DB — ours if we won, or the equal/newer stored one — and ok=false when it cannot
// converge (row vanished, parse failure, account mismatch, or repeated CAS contention).
func casApplyRotation(channelID int, accountUUID string, res *ClaudeOAuthTokenResult, expiresAt, version int64, lastRefresh string) (string, bool) {
	const maxAttempts = 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		ch, err := model.GetChannelById(channelID, true)
		if err != nil || ch == nil {
			return "", false
		}
		cur, err := ParseClaudeOAuthKey(ch.Key)
		if err != nil {
			return "", false
		}
		if accountUUID != "" && strings.TrimSpace(cur.AccountUUID) != accountUUID {
			return "", false
		}
		if cur.TokenVersion >= version {
			// An equal-or-newer rotation is already stored; converge to it (no downgrade).
			return cur.AccessToken, true
		}
		newKey, err := MergeRefreshedTokens(ch.Key, res.AccessToken, res.RefreshToken, expiresAt, version, lastRefresh)
		if err != nil {
			return "", false
		}
		swapped, err := model.CompareAndSwapChannelKey(channelID, ch.Key, newKey)
		if err != nil {
			return "", false
		}
		if swapped {
			return res.AccessToken, true
		}
	}
	return "", false
}

// writeClaudeSiblings fans the rotated token out to sibling OAuth channels (same
// account_uuid) via the same bounded, version-guarded CAS-merge used for the primary.
// The caller holds the account lock, so a CAS miss means an out-of-band admin edit; the
// retry converges and never downgrades a sibling already at an equal/newer version.
// Failures are logged, never aborting the primary refresh.
func writeClaudeSiblings(primaryID int, accountUUID string, res *ClaudeOAuthTokenResult, expiresAt, version int64, lastRefresh string) {
	channels, err := model.GetChannelsByTypeWithKey(constant.ChannelTypeAnthropic)
	if err != nil {
		common.SysLog(fmt.Sprintf("claude oauth: sibling write-back scan failed: %v", err))
		return
	}
	for _, sib := range channels {
		if sib == nil || sib.Id == primaryID {
			continue
		}
		sibSettings := sib.GetOtherSettings()
		if !sibSettings.IsOAuthMode() {
			continue
		}
		sibKey, err := ParseClaudeOAuthKey(sib.Key)
		if err != nil || strings.TrimSpace(sibKey.AccountUUID) != accountUUID {
			continue
		}
		if _, ok := casApplyRotation(sib.Id, accountUUID, res, expiresAt, version, lastRefresh); !ok {
			common.SysLog(fmt.Sprintf("claude oauth: sibling write-back did not converge for channel %d", sib.Id))
		}
	}
}

// ensureClaudeIdentity fills account_uuid / org_uuid / email / type into the key blob
// when they are absent or empty (e.g. a credential imported before the response carried
// them). It never overwrites a present, non-empty value.
func ensureClaudeIdentity(raw string, res *ClaudeOAuthTokenResult) (string, error) {
	if res == nil {
		return raw, nil
	}
	m := map[string]json.RawMessage{}
	raw = strings.TrimSpace(raw)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return "", errors.New("claude oauth: cannot patch identity into invalid key json")
		}
	}
	setIfAbsent := func(field, val string) error {
		if val == "" {
			return nil
		}
		if cur, ok := m[field]; ok {
			var s string
			if json.Unmarshal(cur, &s) == nil && strings.TrimSpace(s) != "" {
				return nil
			}
		}
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		m[field] = b
		return nil
	}
	if err := setIfAbsent("account_uuid", res.AccountUUID); err != nil {
		return "", err
	}
	if err := setIfAbsent("org_uuid", res.OrgUUID); err != nil {
		return "", err
	}
	if err := setIfAbsent("email", res.Email); err != nil {
		return "", err
	}
	if err := setIfAbsent("type", "claude"); err != nil {
		return "", err
	}
	out, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// keyedMutex is a refcounted per-key mutex: Lock(key) blocks other Lock(key) callers
// in this process but never blocks a different key, and the entry is freed when no
// caller holds or waits on it.
type keyedMutex struct {
	mu sync.Mutex
	m  map[string]*keyedMutexEntry
}

type keyedMutexEntry struct {
	mu  sync.Mutex
	ref int
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{m: make(map[string]*keyedMutexEntry)}
}

func (k *keyedMutex) Lock(key string) func() {
	k.mu.Lock()
	e, ok := k.m[key]
	if !ok {
		e = &keyedMutexEntry{}
		k.m[key] = e
	}
	e.ref++
	k.mu.Unlock()

	e.mu.Lock()
	return func() {
		e.mu.Unlock()
		k.mu.Lock()
		e.ref--
		if e.ref == 0 {
			delete(k.m, key)
		}
		k.mu.Unlock()
	}
}
