# API Key Batch Group Switching Design

## Context

Users can already change a single API key's group from the API key list by clicking the group column. This is slow when a user has many API keys and many groups. The requested feature is a user-side batch action: select multiple API keys and move them to one target group.

The implementation should not loop over the existing single-key update endpoint. The current user update handler maps missing IP whitelist and blacklist fields to empty slices, so a group-only update can accidentally clear IP restrictions. A dedicated batch endpoint keeps the operation narrow and avoids unrelated field writes.

## Scope

Add a batch group switch action to the user API key list.

In scope:
- Users can select multiple API keys from the current page.
- A batch action bar appears when at least one key is selected.
- Users choose one target group from the same available group list used by single-key group switching.
- Submitting moves all selected keys to the target group.
- Success clears the selection, refreshes the list, and shows a success toast.
- Failure leaves the selection intact and shows the backend error.

Out of scope:
- Selecting every matching key across all pages.
- Old-group to new-group migration.
- Admin-side bulk API key group switching.
- Bulk unbind to no group. User key creation and editing already require a group, so the batch action should keep that invariant.

## Backend Design

Add a user-authenticated endpoint:

`POST /api/v1/keys/batch/group`

Request:

```json
{
  "ids": [1, 2, 3],
  "group_id": 10
}
```

Response:

```json
{
  "updated": 3
}
```

Validation and behavior:
- Reject empty `ids`.
- Reject non-positive IDs.
- Reject `group_id <= 0`; user-side batch switching requires a real target group.
- Deduplicate IDs before processing.
- Verify every requested key belongs to the authenticated user. If any key is missing, deleted, or owned by another user, reject the entire request.
- Load the target group and verify the current user can bind it using the existing group permission logic.
- Perform one batch update scoped by `user_id`, requested IDs, and `deleted_at IS NULL`.
- If the affected row count does not match the deduped ID count, treat it as a conflict and fail the request.
- Invalidate auth cache for the updated keys so routing and billing use the new group immediately.

Repository support:
- Add a repository method for batch group update by user and IDs.
- Add a repository method to list key strings by user and IDs for precise cache invalidation.
- Keep soft-delete filtering consistent with existing API key repository methods.

## Frontend Design

Update `frontend/src/views/user/KeysView.vue`:
- Add a `select` column at the start of the table.
- Add header checkbox for selecting or clearing all visible rows.
- Add row checkboxes.
- Use the existing `useTableSelection` composable.
- Add a compact batch action bar above the table when selection is non-empty.
- Include selected count, "select this page", "clear selection", a searchable group selector, and a submit button.
- Disable submit until a target group is selected.
- Show a loading state while submitting.
- Clear selection after a successful batch update or when selected rows are no longer present after reload.

API client and types:
- Add `batchUpdateGroup(ids: number[], groupId: number)` to `frontend/src/api/keys.ts`.
- Add a response type with `updated: number`.
- Add Chinese and English i18n strings under `keys.batchGroup`.

## Error Handling

Backend errors should be explicit enough for the UI toast:
- Empty selection: bad request.
- Invalid target group: bad request or not found.
- Group not allowed: existing forbidden error.
- Ownership mismatch: forbidden.
- Concurrent deletion or mismatch during update: conflict.

Frontend behavior:
- Use backend error details when present.
- Keep selected IDs on failure so the user can retry or choose another group.
- Refresh after success to reflect group badges and usage context.

## Testing

Backend tests:
- Service test: updates multiple owned keys to the target group.
- Service test: rejects if any requested key is not owned by the user.
- Service test: rejects if the user cannot bind the target group.
- Repository integration test: batch update only affects matching, non-deleted keys.
- Handler test: validates request payload and returns updated count.

Frontend tests:
- API client test verifies `POST /keys/batch/group` payload.
- Keys view test verifies selecting rows shows the batch bar.
- Keys view test verifies submit calls the batch API, clears selection, and reloads on success.
- Keys view test verifies API failure keeps selection and shows an error.

## Acceptance Criteria

- A user can select multiple API keys on the current page and move them to one target group in one action.
- The operation never changes key name, status, quota, expiration, rate limits, or IP restrictions.
- The backend rejects partial ownership or unauthorized group binding without partial updates.
- Auth cache is invalidated for updated keys.
- Relevant backend and frontend tests pass.
