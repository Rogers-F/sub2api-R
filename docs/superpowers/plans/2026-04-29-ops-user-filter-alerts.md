# Ops User Filter Alerts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add multi-customer filtering to the ops dashboard and let ops alert rules target one or more users.

**Architecture:** Add normalized `user_ids` plumbing through handlers, service filters, repository SQL builders, and frontend API params. Dashboard queries use raw data whenever users are selected because current pre-aggregated tables are platform/group scoped. Alert rules persist selected users in existing JSON filters and the evaluator applies those users only to request-derived metrics.

**Tech Stack:** Go 1.26, Gin handlers, PostgreSQL via `database/sql` and `lib/pq`, Vue 3, TypeScript, Vitest.

---

## File Structure

- Modify `backend/internal/service/ops_dashboard_models.go`: add `UserIDs []int64` to dashboard filters.
- Create `backend/internal/service/ops_user_filter.go`: normalize, parse-like coercion helpers for user ID lists used by services and evaluator.
- Modify `backend/internal/handler/admin/ops_dashboard_handler.go`: parse `user_ids` for overview, trend, histogram, distribution, and token stats if routed through dashboard filters.
- Modify `backend/internal/handler/admin/ops_snapshot_v2_handler.go`: parse `user_ids` and include them in cache keys.
- Modify `backend/internal/handler/admin/ops_handler.go`: parse `user_ids` for request details.
- Modify `backend/internal/service/ops_request_details.go`: add `UserIDs []int64` to request detail filters.
- Modify `backend/internal/repository/ops_repo_dashboard.go`: apply `user_ids` in usage/error query builders and force dashboard overview to raw mode when users are selected.
- Modify `backend/internal/repository/ops_repo_request_details.go`: apply `user_ids` to combined request detail queries.
- Modify `backend/internal/service/ops_alert_evaluator_service.go`: parse `filters.user_ids`, apply them to request metrics, and write dimensions/descriptions.
- Modify `frontend/src/api/admin/ops.ts`: add `user_ids` fields to ops query interfaces.
- Modify `frontend/src/views/admin/ops/OpsDashboard.vue`: add selected user state, URL sync, API params, and child prop wiring.
- Modify `frontend/src/views/admin/ops/components/OpsDashboardHeader.vue`: add customer multi-select control.
- Create `frontend/src/views/admin/ops/components/OpsUserMultiSelect.vue`: reusable async multi-select for dashboard and alert editor.
- Modify `frontend/src/views/admin/ops/components/OpsRequestDetailsModal.vue`: pass selected `user_ids`.
- Modify `frontend/src/views/admin/ops/components/OpsAlertRulesCard.vue`: add scoped customer selector for request-derived alert metrics.
- Modify `frontend/src/i18n/locales/en.ts` and `frontend/src/i18n/locales/zh.ts`: add labels and hints.

## Task 1: Backend User ID Filter Plumbing

**Files:**
- Create: `backend/internal/service/ops_user_filter.go`
- Modify: `backend/internal/service/ops_dashboard_models.go`
- Modify: `backend/internal/service/ops_request_details.go`
- Modify: `backend/internal/handler/admin/ops_dashboard_handler.go`
- Modify: `backend/internal/handler/admin/ops_snapshot_v2_handler.go`
- Modify: `backend/internal/handler/admin/ops_handler.go`
- Test: `backend/internal/handler/admin/ops_user_filter_test.go`

- [ ] **Step 1: Write failing handler/helper tests**

```go
package admin

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseOpsUserIDsQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		rawURL  string
		want    []int64
		wantErr bool
	}{
		{name: "comma separated user_ids", rawURL: "/x?user_ids=3,1,3,2", want: []int64{1, 2, 3}},
		{name: "single user_id alias", rawURL: "/x?user_id=7", want: []int64{7}},
		{name: "blank means all users", rawURL: "/x?user_ids=", want: nil},
		{name: "reject invalid", rawURL: "/x?user_ids=1,nope", wantErr: true},
		{name: "reject non-positive", rawURL: "/x?user_ids=0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.rawURL, nil)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req

			got, err := parseOpsUserIDsQuery(c)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 2: Run the new handler test and confirm it fails**

Run: `cd backend && go test ./internal/handler/admin -run TestParseOpsUserIDsQuery -count=1`

Expected: FAIL because `parseOpsUserIDsQuery` does not exist.

- [ ] **Step 3: Add normalized user ID support**

Add `UserIDs []int64` to `OpsDashboardFilter` and `OpsRequestDetailFilter`.

Create `backend/internal/service/ops_user_filter.go`:

```go
package service

import "sort"

func NormalizeOpsUserIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	if len(out) == 0 {
		return nil
	}
	return out
}
```

Add `parseOpsUserIDsQuery` to the admin handler package. It reads comma-separated `user_ids`, repeated `user_ids`, and single `user_id`, rejects malformed positive integers, and returns `service.NormalizeOpsUserIDs(ids)`.

- [ ] **Step 4: Wire parsed IDs into handlers**

In each dashboard handler after `group_id` parsing:

```go
userIDs, err := parseOpsUserIDsQuery(c)
if err != nil {
	response.BadRequest(c, err.Error())
	return
}
filter.UserIDs = userIDs
```

In `ListRequestDetails`, set `filter.UserIDs = userIDs` and keep existing `user_id` parsing compatible by letting the helper accept that alias.

- [ ] **Step 5: Run handler tests**

Run: `cd backend && go test ./internal/handler/admin -run 'TestParseOpsUserIDsQuery|TestOpsSystemLogHandler' -count=1`

Expected: PASS.

## Task 2: Repository Filtering And Snapshot Cache

**Files:**
- Modify: `backend/internal/repository/ops_repo_dashboard.go`
- Modify: `backend/internal/repository/ops_repo_request_details.go`
- Modify: `backend/internal/handler/admin/ops_snapshot_v2_handler.go`
- Test: `backend/internal/repository/ops_repo_user_filter_test.go`

- [ ] **Step 1: Write failing repository query-builder tests**

```go
package repository

import (
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildUsageWhere_UserIDs(t *testing.T) {
	start := time.Unix(100, 0).UTC()
	end := time.Unix(200, 0).UTC()
	_, where, args, next := buildUsageWhere(&service.OpsDashboardFilter{UserIDs: []int64{20, 10}}, start, end, 1)

	require.Contains(t, where, "ul.user_id = ANY($3)")
	require.Len(t, args, 3)
	require.Equal(t, 4, next)
	valuer, ok := args[2].(driver.Valuer)
	require.True(t, ok)
	value, err := valuer.Value()
	require.NoError(t, err)
	require.Equal(t, "{10,20}", value)
}

func TestBuildErrorWhere_UserIDs(t *testing.T) {
	start := time.Unix(100, 0).UTC()
	end := time.Unix(200, 0).UTC()
	where, args, next := buildErrorWhere(&service.OpsDashboardFilter{UserIDs: []int64{3, 1}}, start, end, 1)

	require.Contains(t, where, "user_id = ANY($3)")
	require.Contains(t, where, "is_count_tokens = FALSE")
	require.Len(t, args, 3)
	require.Equal(t, 4, next)
	valuer, ok := args[2].(driver.Valuer)
	require.True(t, ok)
	value, err := valuer.Value()
	require.NoError(t, err)
	require.Equal(t, "{1,3}", value)
}

func TestSnapshotCacheKeyIncludesUserIDs(t *testing.T) {
	keyA := opsDashboardSnapshotV2CacheKey{UserIDs: []int64{1, 2}}
	keyB := opsDashboardSnapshotV2CacheKey{UserIDs: []int64{2, 3}}

	require.NotEqual(t, keyA.UserIDs, keyB.UserIDs)
	require.False(t, strings.EqualFold(strings.TrimSpace("1,2"), strings.TrimSpace("2,3")))
}
```

- [ ] **Step 2: Run repository tests and confirm they fail**

Run: `cd backend && go test ./internal/repository -run 'TestBuildUsageWhere_UserIDs|TestBuildErrorWhere_UserIDs|TestSnapshotCacheKeyIncludesUserIDs' -count=1`

Expected: FAIL because SQL builders and snapshot key do not include user IDs.

- [ ] **Step 3: Implement repository filters**

In `ops_repo_dashboard.go`, import `github.com/lib/pq`. In `buildUsageWhere` and `buildErrorWhere`, normalize user IDs and append:

```go
userIDs := service.NormalizeOpsUserIDs(filter.UserIDs)
if len(userIDs) > 0 {
	args = append(args, pq.Array(userIDs))
	clauses = append(clauses, fmt.Sprintf("ul.user_id = ANY($%d)", idx))
	idx++
}
```

For `buildErrorWhere`, use `user_id = ANY($%d)`.

In `GetDashboardOverview`, before the query mode switch:

```go
if len(service.NormalizeOpsUserIDs(filter.UserIDs)) > 0 {
	return r.getDashboardOverviewRaw(ctx, filter)
}
```

In `ops_repo_request_details.go`, import `github.com/lib/pq` and add:

```go
if userIDs := service.NormalizeOpsUserIDs(filter.UserIDs); len(userIDs) > 0 {
	addCondition(fmt.Sprintf("user_id = ANY($%d)", len(args)+1), pq.Array(userIDs))
}
```

- [ ] **Step 4: Include user IDs in snapshot cache key**

Add `UserIDs []int64` to `opsDashboardSnapshotV2CacheKey`. Set it with `service.NormalizeOpsUserIDs(filter.UserIDs)` where the key is built.

- [ ] **Step 5: Run repository tests**

Run: `cd backend && go test ./internal/repository -run 'TestBuildUsageWhere_UserIDs|TestBuildErrorWhere_UserIDs|TestSnapshotCacheKeyIncludesUserIDs' -count=1`

Expected: PASS.

## Task 3: User-Scoped Alert Evaluation

**Files:**
- Modify: `backend/internal/service/ops_alert_evaluator_service.go`
- Test: `backend/internal/service/ops_alert_evaluator_service_test.go`

- [ ] **Step 1: Write failing evaluator tests**

Add tests for scope parsing and metric filter forwarding:

```go
func TestParseOpsAlertRuleScope_UserIDs(t *testing.T) {
	platform, groupID, region, userIDs := parseOpsAlertRuleScope(map[string]any{
		"platform": "openai",
		"group_id": float64(9),
		"region": "us",
		"user_ids": []any{float64(3), "1", int64(2), float64(3)},
	})

	require.Equal(t, "openai", platform)
	require.NotNil(t, groupID)
	require.Equal(t, int64(9), *groupID)
	require.NotNil(t, region)
	require.Equal(t, "us", *region)
	require.Equal(t, []int64{1, 2, 3}, userIDs)
}

func TestComputeRuleMetricPassesUserIDsForRequestMetrics(t *testing.T) {
	var captured []int64
	repo := &stubOpsRepo{overview: &OpsDashboardOverview{
		RequestCountSLA: 10,
		ErrorRate: 0.2,
	}}
	repo.GetDashboardOverviewFn = func(_ context.Context, filter *OpsDashboardFilter) (*OpsDashboardOverview, error) {
		captured = append([]int64{}, filter.UserIDs...)
		return repo.overview, nil
	}

	svc := &OpsAlertEvaluatorService{opsRepo: repo}
	got, ok := svc.computeRuleMetric(
		context.Background(),
		&OpsAlertRule{MetricType: "error_rate"},
		nil,
		time.Now().Add(-time.Minute),
		time.Now(),
		"",
		nil,
		[]int64{8, 4},
	)

	require.True(t, ok)
	require.InDelta(t, 20.0, got, 0.0001)
	require.Equal(t, []int64{4, 8}, captured)
}
```

Update `stubOpsRepo` with optional `GetDashboardOverviewFn` before this test can pass.

- [ ] **Step 2: Run evaluator tests and confirm they fail**

Run: `cd backend && go test -tags unit ./internal/service -run 'TestParseOpsAlertRuleScope_UserIDs|TestComputeRuleMetricPassesUserIDsForRequestMetrics' -count=1`

Expected: FAIL because alert scope does not parse `user_ids` and `computeRuleMetric` has no user ID argument.

- [ ] **Step 3: Extend scope and metric evaluation**

Change `parseOpsAlertRuleScope` to return `(platform string, groupID *int64, region *string, userIDs []int64)`. Parse `filters["user_ids"]` from `[]any`, `[]int64`, `[]float64`, `[]string`, or comma-separated string. Normalize with `NormalizeOpsUserIDs`.

Change `computeRuleMetric` signature to:

```go
func (s *OpsAlertEvaluatorService) computeRuleMetric(
	ctx context.Context,
	rule *OpsAlertRule,
	systemMetrics *OpsSystemMetricsSnapshot,
	start time.Time,
	end time.Time,
	platform string,
	groupID *int64,
	userIDs []int64,
) (float64, bool)
```

When building the dashboard overview filter, set `UserIDs: NormalizeOpsUserIDs(userIDs)`.

- [ ] **Step 4: Add dimensions and descriptions**

Change `buildOpsAlertDimensions` and `buildOpsAlertDescription` to accept `userIDs []int64`. Add `user_ids` to dimensions only when non-empty. Append `user_ids=1,2` to the scope text in descriptions.

- [ ] **Step 5: Run evaluator tests**

Run: `cd backend && go test -tags unit ./internal/service -run 'TestParseOpsAlertRuleScope_UserIDs|TestComputeRuleMetricPassesUserIDsForRequestMetrics|TestComputeRuleMetricNewIndicators' -count=1`

Expected: PASS.

## Task 4: Frontend Dashboard State And Multi-Select

**Files:**
- Create: `frontend/src/views/admin/ops/components/OpsUserMultiSelect.vue`
- Modify: `frontend/src/api/admin/ops.ts`
- Modify: `frontend/src/views/admin/ops/OpsDashboard.vue`
- Modify: `frontend/src/views/admin/ops/components/OpsDashboardHeader.vue`
- Modify: `frontend/src/views/admin/ops/components/OpsRequestDetailsModal.vue`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Test: `frontend/src/views/admin/ops/components/__tests__/OpsUserMultiSelect.spec.ts`

- [ ] **Step 1: Write failing frontend component test**

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import OpsUserMultiSelect from '../OpsUserMultiSelect.vue'

vi.mock('@/api/admin/usage', () => ({
  searchUsers: vi.fn().mockResolvedValue([
    { id: 1, email: 'a@example.com' },
    { id: 2, email: 'b@example.com' }
  ])
}))

describe('OpsUserMultiSelect', () => {
  it('emits selected user IDs and clears them', async () => {
    const wrapper = mount(OpsUserMultiSelect, {
      props: { modelValue: [] },
      global: {
        mocks: { $t: (key: string) => key },
        stubs: { Icon: true }
      }
    })

    await wrapper.find('button[data-testid="ops-user-filter-trigger"]').trigger('click')
    await Promise.resolve()
    await Promise.resolve()

    await wrapper.find('input[type="checkbox"][value="1"]').setValue(true)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([1])

    await wrapper.setProps({ modelValue: [1] })
    await wrapper.find('button[data-testid="ops-user-filter-clear"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([])
  })
})
```

- [ ] **Step 2: Run the frontend test and confirm it fails**

Run: `cd frontend && pnpm vitest run src/views/admin/ops/components/__tests__/OpsUserMultiSelect.spec.ts`

Expected: FAIL because `OpsUserMultiSelect.vue` does not exist.

- [ ] **Step 3: Implement `OpsUserMultiSelect.vue`**

Create a focused popover component with props `modelValue: number[]`, emits `update:modelValue`, imports `searchUsers` from `@/api/admin/usage`, loads options on open and search input changes, renders checkbox rows, and keeps selected IDs sorted.

Use stable test hooks:

```vue
<button data-testid="ops-user-filter-trigger" type="button" @click="toggleOpen">...</button>
<button v-if="selectedIds.length" data-testid="ops-user-filter-clear" type="button" @click.stop="emitSelection([])">...</button>
<input v-model="query" type="search" />
<input :value="user.id" type="checkbox" :checked="selectedSet.has(user.id)" @change="toggleUser(user.id)" />
```

- [ ] **Step 4: Wire dashboard user IDs**

In `OpsDashboard.vue`, add `userIds = ref<number[]>([])`, `QUERY_KEYS.userIds = 'user_ids'`, parse comma-separated values from route, serialize selected IDs back to route, pass `:user-ids="userIds"` to `OpsDashboardHeader` and request details modal, and add `@update:user-ids="onUserIdsChange"`.

Add to `buildApiParams()` and `buildSwitchTrendParams()`:

```ts
const selectedUserIds = normalizeUserIds(userIds.value)
if (selectedUserIds.length > 0) params.user_ids = selectedUserIds.join(',')
```

Update watchers to include `userIds.value.join(',')`.

- [ ] **Step 5: Wire header and request details modal**

In `OpsDashboardHeader.vue`, add prop `userIds: number[]` and emit `update:userIds`. Place `OpsUserMultiSelect` next to platform/group filters.

In `OpsRequestDetailsModal.vue`, add prop `userIds?: number[]`, add it to the data watch list, and set `params.user_ids = selected.join(',')` when non-empty.

Update `OpsRequestDetailsParams` and dashboard API param types in `ops.ts` with `user_ids?: string`.

- [ ] **Step 6: Add i18n keys**

Add English and Chinese keys under `admin.ops`:

```ts
customers: 'Customers',
allCustomers: 'All customers',
searchCustomers: 'Search customers',
selectedCustomers: '{count} selected',
clearCustomers: 'Clear customers',
customerFilterLoadFailed: 'Failed to load customers'
```

Use matching Chinese text in `zh.ts`.

- [ ] **Step 7: Run frontend targeted test**

Run: `cd frontend && pnpm vitest run src/views/admin/ops/components/__tests__/OpsUserMultiSelect.spec.ts`

Expected: PASS.

## Task 5: Frontend Alert Rule Editor User Scope

**Files:**
- Modify: `frontend/src/views/admin/ops/components/OpsAlertRulesCard.vue`
- Test: manual through typecheck because this component has no existing focused test harness.

- [ ] **Step 1: Add request-metric detection**

Add:

```ts
const requestMetricTypes = new Set<MetricType>(['success_rate', 'error_rate', 'upstream_error_rate'])

const isUserScopeAllowed = computed(() => {
  const metricType = draft.value?.metric_type
  return metricType ? requestMetricTypes.has(metricType) : false
})
```

- [ ] **Step 2: Add `draftUserIds` computed setter**

```ts
const draftUserIds = computed<number[]>({
  get() {
    const raw = draft.value?.filters?.user_ids
    if (!Array.isArray(raw)) return []
    return normalizeUserIds(raw.map((v) => Number(v)))
  },
  set(value) {
    if (!draft.value) return
    const ids = normalizeUserIds(value)
    if (ids.length === 0) {
      if (draft.value.filters) delete draft.value.filters.user_ids
    } else {
      if (!draft.value.filters) draft.value.filters = {}
      draft.value.filters.user_ids = ids
    }
    if (draft.value.filters && Object.keys(draft.value.filters).length === 0) {
      delete draft.value.filters
    }
  }
})
```

- [ ] **Step 3: Render selector for request metrics**

Add `OpsUserMultiSelect v-model="draftUserIds"` under the group filter, only when `isUserScopeAllowed` is true. When metric changes away from a request metric, clear `draftUserIds`.

- [ ] **Step 4: Preserve save behavior**

Before create/update, ensure empty `filters.user_ids` is removed and non-empty IDs are normalized. Continue sending the whole `draft.value` object so the existing backend JSON filter behavior persists the new scope.

- [ ] **Step 5: Run typecheck**

Run: `cd frontend && pnpm typecheck`

Expected: PASS.

## Task 6: Full Verification

**Files:**
- No new source files unless previous tasks expose integration gaps.

- [ ] **Step 1: Run backend focused checks**

Run:

```bash
cd backend
go test ./internal/handler/admin -run TestParseOpsUserIDsQuery -count=1
go test ./internal/repository -run 'TestBuildUsageWhere_UserIDs|TestBuildErrorWhere_UserIDs|TestSnapshotCacheKeyIncludesUserIDs' -count=1
go test -tags unit ./internal/service -run 'TestParseOpsAlertRuleScope_UserIDs|TestComputeRuleMetricPassesUserIDsForRequestMetrics|TestComputeRuleMetricNewIndicators' -count=1
```

Expected: all PASS.

- [ ] **Step 2: Run frontend focused checks**

Run:

```bash
cd frontend
pnpm vitest run src/views/admin/ops/components/__tests__/OpsUserMultiSelect.spec.ts
pnpm typecheck
```

Expected: all PASS.

- [ ] **Step 3: Run final status check**

Run: `git status --short`

Expected: only intended source, test, i18n, and plan files are modified.
