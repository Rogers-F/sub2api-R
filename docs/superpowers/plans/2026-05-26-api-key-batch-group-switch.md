# API Key Batch Group Switching Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a user-side batch action that moves selected API keys to one target group without changing any other API key fields.

**Architecture:** Add a dedicated user endpoint for batch group switching instead of looping through the existing single-key update endpoint. The service validates ownership and group permissions up front, performs one scoped repository update, and invalidates auth cache for the moved keys. The frontend adds table selection and a compact batch action bar that calls the new endpoint.

**Tech Stack:** Go 1.26, Gin, Ent, Vue 3, TypeScript, Vitest, vue-test-utils.

---

### Task 1: Backend Service and Repository

**Files:**
- Modify: `backend/internal/service/api_key_service.go`
- Modify: `backend/internal/service/api_key.go`
- Modify: `backend/internal/repository/api_key_repo.go`
- Test: `backend/internal/service/api_key_service_batch_group_test.go`
- Update compile stubs in tests that implement `service.APIKeyRepository`

- [ ] **Step 1: Write the failing service tests**

Create `backend/internal/service/api_key_service_batch_group_test.go` with tests for success, ownership mismatch, and unauthorized target group.

```go
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
		apiKeyRepo: repo,
		userRepo: userRepo,
		groupRepo: groupRepo,
		userSubRepo: &userSubRepoNoop{},
		cache: cache,
	}

	updated, err := svc.BatchUpdateGroup(context.Background(), 7, BatchUpdateAPIKeyGroupRequest{
		IDs: []int64{1, 2, 2},
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
		apiKeyRepo: repo,
		userRepo: &batchGroupUserRepoStub{user: &User{ID: 7, AllowedGroups: []int64{10}}},
		groupRepo: &batchGroupGroupRepoStub{group: &Group{ID: 10, Status: StatusActive}},
		userSubRepo: &userSubRepoNoop{},
	}

	_, err := svc.BatchUpdateGroup(context.Background(), 7, BatchUpdateAPIKeyGroupRequest{
		IDs: []int64{1, 2},
		GroupID: 10,
	})

	require.ErrorIs(t, err, ErrInsufficientPerms)
	require.Empty(t, repo.updatedIDs)
}

func TestAPIKeyService_BatchUpdateGroup_RejectsGroupNotAllowed(t *testing.T) {
	repo := &batchGroupAPIKeyRepoStub{ownedIDs: []int64{1, 2}}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo: &batchGroupUserRepoStub{user: &User{ID: 7, AllowedGroups: []int64{9}}},
		groupRepo: &batchGroupGroupRepoStub{group: &Group{ID: 10, Status: StatusActive, IsExclusive: true}},
		userSubRepo: &userSubRepoNoop{},
	}

	_, err := svc.BatchUpdateGroup(context.Background(), 7, BatchUpdateAPIKeyGroupRequest{
		IDs: []int64{1, 2},
		GroupID: 10,
	})

	require.ErrorIs(t, err, ErrGroupNotAllowed)
	require.Empty(t, repo.updatedIDs)
}
```

- [ ] **Step 2: Run the service tests and verify they fail**

Run:

```bash
cd backend && go test -tags unit ./internal/service -run 'TestAPIKeyService_BatchUpdateGroup' -count=1
```

Expected: fail because `BatchUpdateAPIKeyGroupRequest`, `BatchUpdateGroup`, and new repository methods do not exist.

- [ ] **Step 3: Add service request, errors, and repository interface methods**

Add to `backend/internal/service/api_key.go`:

```go
type BatchUpdateAPIKeyGroupRequest struct {
	IDs     []int64 `json:"ids"`
	GroupID int64   `json:"group_id"`
}
```

Add to `backend/internal/service/api_key_service.go`:

```go
var (
	ErrNoAPIKeysSelected       = infraerrors.BadRequest("NO_API_KEYS_SELECTED", "no api keys selected")
	ErrInvalidAPIKeyID         = infraerrors.BadRequest("INVALID_API_KEY_ID", "api key id must be positive")
	ErrInvalidAPIKeyGroupID    = infraerrors.BadRequest("INVALID_API_KEY_GROUP_ID", "group_id must be positive")
	ErrAPIKeyBatchUpdateFailed = infraerrors.Conflict("API_KEY_BATCH_UPDATE_FAILED", "api key batch update did not update all requested keys")
)
```

Extend `APIKeyRepository`:

```go
	BatchUpdateGroupIDByUserAndIDs(ctx context.Context, userID int64, ids []int64, groupID int64) (int64, error)
	ListKeysByUserAndIDs(ctx context.Context, userID int64, ids []int64) ([]string, error)
```

- [ ] **Step 4: Implement minimal service behavior**

Add to `backend/internal/service/api_key_service.go`:

```go
func uniquePositiveAPIKeyIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, ErrNoAPIKeysSelected
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, ErrInvalidAPIKeyID
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, ErrNoAPIKeysSelected
	}
	return out, nil
}

func (s *APIKeyService) BatchUpdateGroup(ctx context.Context, userID int64, req BatchUpdateAPIKeyGroupRequest) (int64, error) {
	if req.GroupID <= 0 {
		return 0, ErrInvalidAPIKeyGroupID
	}
	ids, err := uniquePositiveAPIKeyIDs(req.IDs)
	if err != nil {
		return 0, err
	}

	validIDs, err := s.apiKeyRepo.VerifyOwnership(ctx, userID, ids)
	if err != nil {
		return 0, fmt.Errorf("verify api key ownership: %w", err)
	}
	if len(validIDs) != len(ids) {
		return 0, ErrInsufficientPerms
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get user: %w", err)
	}
	group, err := s.groupRepo.GetByID(ctx, req.GroupID)
	if err != nil {
		return 0, fmt.Errorf("get group: %w", err)
	}
	if !s.canUserBindGroup(ctx, user, group) {
		return 0, ErrGroupNotAllowed
	}

	keys, err := s.apiKeyRepo.ListKeysByUserAndIDs(ctx, userID, ids)
	if err != nil {
		return 0, fmt.Errorf("list api keys for cache invalidation: %w", err)
	}
	updated, err := s.apiKeyRepo.BatchUpdateGroupIDByUserAndIDs(ctx, userID, ids, req.GroupID)
	if err != nil {
		return 0, fmt.Errorf("batch update api key group: %w", err)
	}
	if updated != int64(len(ids)) {
		return 0, ErrAPIKeyBatchUpdateFailed
	}
	for _, key := range keys {
		s.InvalidateAuthCacheByKey(ctx, key)
	}
	return updated, nil
}
```

- [ ] **Step 5: Implement repository methods**

Add to `backend/internal/repository/api_key_repo.go`:

```go
func (r *apiKeyRepository) BatchUpdateGroupIDByUserAndIDs(ctx context.Context, userID int64, ids []int64, groupID int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	client := clientFromContext(ctx, r.client)
	n, err := client.APIKey.Update().
		Where(apikey.UserIDEQ(userID), apikey.IDIn(ids...), apikey.DeletedAtIsNil()).
		SetGroupID(groupID).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return int64(n), err
}

func (r *apiKeyRepository) ListKeysByUserAndIDs(ctx context.Context, userID int64, ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	return r.activeQuery().
		Where(apikey.UserIDEQ(userID), apikey.IDIn(ids...)).
		Select(apikey.FieldKey).
		Strings(ctx)
}
```

- [ ] **Step 6: Update APIKeyRepository stubs**

Run:

```bash
rg -n "func \\(.*\\) ListKeysByUserID|func \\(.*\\) UpdateGroupIDByUserAndGroup|APIKeyRepository" backend/internal -g '*_test.go'
```

For each stub that implements `APIKeyRepository`, add:

```go
func (s *stubType) BatchUpdateGroupIDByUserAndIDs(context.Context, int64, []int64, int64) (int64, error) {
	panic("unexpected BatchUpdateGroupIDByUserAndIDs call")
}

func (s *stubType) ListKeysByUserAndIDs(context.Context, int64, []int64) ([]string, error) {
	panic("unexpected ListKeysByUserAndIDs call")
}
```

For the new batch service test stub, implement the methods with captured arguments instead of panics.

- [ ] **Step 7: Run the service tests and verify they pass**

Run:

```bash
cd backend && go test -tags unit ./internal/service -run 'TestAPIKeyService_BatchUpdateGroup' -count=1
```

Expected: pass.

### Task 2: Backend Handler and Route

**Files:**
- Modify: `backend/internal/handler/api_key_handler.go`
- Modify: `backend/internal/server/routes/user.go`
- Test: `backend/internal/handler/api_key_handler_batch_group_test.go`

- [ ] **Step 1: Write failing handler tests**

Create `backend/internal/handler/api_key_handler_batch_group_test.go` with a Gin router that injects `middleware.AuthSubject{UserID: 7}` and calls the new handler.

```go
func TestAPIKeyHandler_BatchUpdateGroup_Success(t *testing.T) {
	router, svc := setupBatchAPIKeyHandlerTest()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/batch/group", bytes.NewBufferString(`{"ids":[1,2],"group_id":10}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), svc.lastUserID)
	require.Equal(t, []int64{1, 2}, svc.lastReq.IDs)
	require.Equal(t, int64(10), svc.lastReq.GroupID)
	require.Contains(t, rec.Body.String(), `"updated":2`)
}
```

- [ ] **Step 2: Run handler tests and verify they fail**

Run:

```bash
cd backend && go test ./internal/handler -run 'TestAPIKeyHandler_BatchUpdateGroup' -count=1
```

Expected: fail because `BatchUpdateGroup` handler does not exist.

- [ ] **Step 3: Add handler request and method**

Add to `backend/internal/handler/api_key_handler.go`:

```go
type BatchUpdateAPIKeyGroupRequest struct {
	IDs     []int64 `json:"ids" binding:"required"`
	GroupID int64   `json:"group_id" binding:"required"`
}

func (h *APIKeyHandler) BatchUpdateGroup(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req BatchUpdateAPIKeyGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	updated, err := h.apiKeyService.BatchUpdateGroup(c.Request.Context(), subject.UserID, service.BatchUpdateAPIKeyGroupRequest{
		IDs: req.IDs,
		GroupID: req.GroupID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"updated": updated})
}
```

- [ ] **Step 4: Register the route**

Add before `keys.GET("/:id", ...)` in `backend/internal/server/routes/user.go`:

```go
			keys.POST("/batch/group", h.APIKey.BatchUpdateGroup)
```

- [ ] **Step 5: Run handler tests**

Run:

```bash
cd backend && go test ./internal/handler -run 'TestAPIKeyHandler_BatchUpdateGroup' -count=1
```

Expected: pass.

### Task 3: Frontend API Client

**Files:**
- Modify: `frontend/src/api/keys.ts`
- Modify: `frontend/src/types/index.ts`
- Test: `frontend/src/api/__tests__/keys.spec.ts`

- [ ] **Step 1: Write failing API client test**

Create `frontend/src/api/__tests__/keys.spec.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('@/i18n', () => ({ getLocale: () => 'zh-CN' }))

describe('keysAPI', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
  })

  it('posts selected key ids and target group id for batch group update', async () => {
    const { apiClient } = await import('@/api/client')
    const adapter = vi.fn().mockResolvedValue({
      status: 200,
      data: { code: 0, data: { updated: 2 } },
      headers: {},
      config: {},
      statusText: 'OK',
    })
    apiClient.defaults.adapter = adapter
    const { keysAPI } = await import('@/api/keys')

    await expect(keysAPI.batchUpdateGroup([1, 2], 10)).resolves.toEqual({ updated: 2 })
    expect(adapter).toHaveBeenCalledWith(expect.objectContaining({
      method: 'post',
      url: '/keys/batch/group',
      data: JSON.stringify({ ids: [1, 2], group_id: 10 }),
    }))
  })
})
```

- [ ] **Step 2: Run API client test and verify it fails**

Run:

```bash
cd frontend && pnpm vitest run src/api/__tests__/keys.spec.ts
```

Expected: fail because `batchUpdateGroup` does not exist.

- [ ] **Step 3: Add type and client method**

Add to `frontend/src/types/index.ts`:

```ts
export interface BatchUpdateApiKeyGroupRequest {
  ids: number[]
  group_id: number
}

export interface BatchUpdateApiKeyGroupResponse {
  updated: number
}
```

Update `frontend/src/api/keys.ts`:

```ts
import type {
  ApiKey,
  CreateApiKeyRequest,
  UpdateApiKeyRequest,
  PaginatedResponse,
  BatchUpdateApiKeyGroupResponse
} from '@/types'

export async function batchUpdateGroup(ids: number[], groupId: number): Promise<BatchUpdateApiKeyGroupResponse> {
  const { data } = await apiClient.post<BatchUpdateApiKeyGroupResponse>('/keys/batch/group', {
    ids,
    group_id: groupId
  })
  return data
}

export const keysAPI = {
  list,
  getById,
  create,
  update,
  delete: deleteKey,
  toggleStatus,
  batchUpdateGroup
}
```

- [ ] **Step 4: Run API client test**

Run:

```bash
cd frontend && pnpm vitest run src/api/__tests__/keys.spec.ts
```

Expected: pass.

### Task 4: Frontend Batch Selection UI

**Files:**
- Modify: `frontend/src/views/user/KeysView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/views/user/__tests__/KeysView.batchGroup.spec.ts`

- [ ] **Step 1: Write failing KeysView tests**

Create `frontend/src/views/user/__tests__/KeysView.batchGroup.spec.ts` with mocked `keysAPI`, `userGroupsAPI`, `usageAPI`, and app store. The tests should mount `KeysView.vue` with simple stubs for layout and table dependencies, then assert that selecting a row shows the batch bar and submitting calls `keysAPI.batchUpdateGroup`.

```ts
it('shows batch group bar after selecting a key', async () => {
  const wrapper = await mountKeysView()

  await wrapper.find('[data-test="key-row-select-1"]').setValue(true)

  expect(wrapper.find('[data-test="keys-batch-group-bar"]').exists()).toBe(true)
  expect(wrapper.text()).toContain('已选择 1 个密钥')
})

it('submits selected ids and target group id', async () => {
  const wrapper = await mountKeysView()

  await wrapper.find('[data-test="key-row-select-1"]').setValue(true)
  await wrapper.find('[data-test="batch-group-select"]').setValue('10')
  await wrapper.find('[data-test="batch-group-submit"]').trigger('click')
  await flushPromises()

  expect(mockBatchUpdateGroup).toHaveBeenCalledWith([1], 10)
  expect(mockShowSuccess).toHaveBeenCalled()
})
```

- [ ] **Step 2: Run KeysView tests and verify they fail**

Run:

```bash
cd frontend && pnpm vitest run src/views/user/__tests__/KeysView.batchGroup.spec.ts
```

Expected: fail because batch selection UI does not exist.

- [ ] **Step 3: Add i18n strings**

Add under `keys` in `frontend/src/i18n/locales/zh.ts`:

```ts
    batchGroup: {
      selected: '已选择 {count} 个密钥',
      selectCurrentPage: '本页全选',
      clear: '清除选择',
      targetGroup: '目标分组',
      selectTargetGroup: '选择目标分组',
      submit: '批量切换分组',
      submitting: '正在切换...',
      success: '已切换 {count} 个密钥的分组',
      failed: '批量切换分组失败'
    },
```

Add under `keys` in `frontend/src/i18n/locales/en.ts`:

```ts
    batchGroup: {
      selected: '{count} key(s) selected',
      selectCurrentPage: 'Select this page',
      clear: 'Clear selection',
      targetGroup: 'Target group',
      selectTargetGroup: 'Select target group',
      submit: 'Batch switch group',
      submitting: 'Switching...',
      success: 'Switched group for {count} key(s)',
      failed: 'Failed to batch switch group'
    },
```

- [ ] **Step 4: Add selection state and computed columns**

Update `frontend/src/views/user/KeysView.vue` imports:

```ts
import { ref, computed, onMounted, onUnmounted, watch, type ComponentPublicInstance } from 'vue'
import { useTableSelection } from '@/composables/useTableSelection'
```

Add state:

```ts
const batchGroupId = ref<number | null>(null)
const batchGroupSubmitting = ref(false)

const {
  selectedIds,
  allVisibleSelected,
  toggle: toggleSelectedKey,
  clear: clearSelectedKeys,
  toggleVisible,
  selectVisible
} = useTableSelection<ApiKey>({
  rows: apiKeys,
  getId: (key) => key.id
})

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', sortable: false },
  { key: 'name', label: t('common.name'), sortable: true },
  { key: 'key', label: t('keys.apiKey'), sortable: false },
  { key: 'group', label: t('keys.group'), sortable: false },
  { key: 'usage', label: t('keys.usage'), sortable: false },
  { key: 'rate_limit', label: t('keys.rateLimitColumn'), sortable: false },
  { key: 'expires_at', label: t('keys.expiresAt'), sortable: true },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'last_used_at', label: t('keys.lastUsedAt'), sortable: true },
  { key: 'created_at', label: t('keys.created'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false }
])
```

Add a watcher after `loadApiKeys` state:

```ts
watch(apiKeys, (rows) => {
  const visible = new Set(rows.map((key) => key.id))
  const stale = selectedIds.value.filter((id) => !visible.has(id))
  if (stale.length > 0) {
    clearSelectedKeys()
  }
})
```

- [ ] **Step 5: Add template controls**

Add before `<DataTable>`:

```vue
<div
  v-if="selectedIds.length > 0"
  data-test="keys-batch-group-bar"
  class="mb-4 flex flex-col gap-3 rounded-lg bg-accent-50 p-3 dark:bg-accent-800/30 lg:flex-row lg:items-center lg:justify-between"
>
  <div class="flex flex-wrap items-center gap-2">
    <span class="text-sm font-medium text-primary-900 dark:text-primary-100">
      {{ t('keys.batchGroup.selected', { count: selectedIds.length }) }}
    </span>
    <button type="button" class="text-xs font-medium text-primary-700" @click="selectVisible">
      {{ t('keys.batchGroup.selectCurrentPage') }}
    </button>
    <button type="button" class="text-xs font-medium text-primary-700" @click="clearSelectedKeys">
      {{ t('keys.batchGroup.clear') }}
    </button>
  </div>
  <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
    <Select
      v-model="batchGroupId"
      data-test="batch-group-select"
      class="min-w-64"
      :options="groupOptions"
      :placeholder="t('keys.batchGroup.selectTargetGroup')"
      :searchable="true"
      :search-placeholder="t('keys.searchGroup')"
    />
    <button
      type="button"
      data-test="batch-group-submit"
      class="btn btn-primary"
      :disabled="batchGroupSubmitting || batchGroupId === null"
      @click="batchUpdateSelectedGroup"
    >
      {{ batchGroupSubmitting ? t('keys.batchGroup.submitting') : t('keys.batchGroup.submit') }}
    </button>
  </div>
</div>
```

Add DataTable slots:

```vue
<template #header-select>
  <input
    type="checkbox"
    class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
    :checked="allVisibleSelected"
    @click.stop
    @change="toggleSelectAllVisible($event)"
  />
</template>

<template #cell-select="{ row }">
  <input
    type="checkbox"
    :data-test="`key-row-select-${row.id}`"
    class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
    :checked="selectedIds.includes(row.id)"
    @change="toggleSelectedKey(row.id)"
  />
</template>
```

- [ ] **Step 6: Add submit handlers**

Add to `KeysView.vue`:

```ts
const toggleSelectAllVisible = (event: Event) => {
  const target = event.target as HTMLInputElement
  toggleVisible(target.checked)
}

const batchUpdateSelectedGroup = async () => {
  if (selectedIds.value.length === 0 || batchGroupId.value === null) return
  batchGroupSubmitting.value = true
  try {
    const result = await keysAPI.batchUpdateGroup(selectedIds.value, batchGroupId.value)
    appStore.showSuccess(t('keys.batchGroup.success', { count: result.updated }))
    clearSelectedKeys()
    batchGroupId.value = null
    await loadApiKeys()
  } catch (error: any) {
    appStore.showError(error?.message || error?.response?.data?.detail || t('keys.batchGroup.failed'))
  } finally {
    batchGroupSubmitting.value = false
  }
}
```

- [ ] **Step 7: Run KeysView tests**

Run:

```bash
cd frontend && pnpm vitest run src/views/user/__tests__/KeysView.batchGroup.spec.ts
```

Expected: pass.

### Task 5: Full Verification

**Files:**
- All files changed by Tasks 1-4

- [ ] **Step 1: Run targeted backend tests**

Run:

```bash
cd backend && go test -tags unit ./internal/service -run 'TestAPIKeyService_BatchUpdateGroup' -count=1
cd backend && go test ./internal/handler -run 'TestAPIKeyHandler_BatchUpdateGroup' -count=1
```

Expected: both commands pass.

- [ ] **Step 2: Run targeted frontend tests**

Run:

```bash
cd frontend && pnpm vitest run src/api/__tests__/keys.spec.ts src/views/user/__tests__/KeysView.batchGroup.spec.ts
```

Expected: pass.

- [ ] **Step 3: Run frontend typecheck**

Run:

```bash
cd frontend && pnpm typecheck
```

Expected: pass.

- [ ] **Step 4: Run backend package tests likely affected by interface changes**

Run:

```bash
cd backend && go test -tags unit ./internal/service ./internal/server/middleware -count=1
cd backend && go test ./internal/handler ./internal/repository -run 'APIKey|BatchUpdateGroup' -count=1
```

Expected: pass or report any pre-existing unrelated failures with exact output.

- [ ] **Step 5: Review diff**

Run:

```bash
git diff -- backend/internal/service/api_key.go backend/internal/service/api_key_service.go backend/internal/repository/api_key_repo.go backend/internal/handler/api_key_handler.go backend/internal/server/routes/user.go frontend/src/api/keys.ts frontend/src/types/index.ts frontend/src/views/user/KeysView.vue frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
```

Expected: changes only implement batch group switching and tests.
