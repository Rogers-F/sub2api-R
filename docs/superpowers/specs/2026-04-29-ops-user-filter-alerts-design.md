# Ops User Filter And User-Scoped Alerts Design

## Context

The admin ops dashboard currently supports global filters for platform and group. Ops alert rules already persist free-form `filters`, and the alert evaluator consumes platform and group scope from those filters. Request details already understand `user_id`, and both `usage_logs` and `ops_error_logs` carry user IDs.

The requested feature is to filter ops monitoring by one or more customers and allow alert rules to target one or more customers.

## Goals

- Add a multi-select customer filter to the ops dashboard.
- Keep no selection as the existing "all customers" behavior.
- Apply the customer filter to dashboard overview, throughput trend, error trend, latency histogram, error distribution, and request details.
- Add customer multi-select to alert rule editing.
- Evaluate user-scoped alert rules using only the selected users.
- Store selected customers in alert rule `filters.user_ids`.
- Preserve selected dashboard users in the URL so refresh and shared links keep the same filter.

## Non-Goals

- No customer tags, saved segments, or customer groups.
- No per-user notification recipient routing.
- No changes to billing or usage cleanup flows.
- No migration is needed because alert rule filters are already JSON.

## Product Behavior

The ops dashboard header gets a "Customers" multi-select control. Admins can search by email or username and select multiple users with checkboxes. Selected users appear as compact selected items. Clearing all selections restores the current all-customer dashboard.

Dashboard query state adds `user_ids=1,2,3` in the URL. Invalid, empty, or non-positive IDs are ignored. Route synchronization follows the same pattern as `platform`, `group_id`, and `mode`.

Alert rule editor gets the same customer selector under the existing group filter. For request-derived metrics, selected users narrow the monitored population. For system-only metrics like CPU, memory, and queue depth, user selection is not meaningful and should be hidden or ignored.

## Backend Design

`service.OpsDashboardFilter` gains `UserIDs []int64`.

Admin ops dashboard handlers parse `user_ids` from a comma-separated query parameter and populate the filter. Existing `user_id` can be accepted as a compatibility alias for a single user where convenient, but the dashboard should send `user_ids`.

Repository query builders add user filtering:

- `buildUsageWhere`: add `ul.user_id = ANY($n)` when `UserIDs` is non-empty.
- `buildErrorWhere`: add `user_id = ANY($n)` when `UserIDs` is non-empty.

Pre-aggregated dashboard mode cannot support user-level filtering with the current aggregate tables. When `UserIDs` is non-empty, repository dashboard overview should use raw mode. This preserves correctness over speed for targeted customer drilldowns.

Snapshot cache keys include `UserIDs`, sorted and de-duplicated.

Alert evaluator changes:

- Parse `filters.user_ids` from JSON into `[]int64`.
- Include the selected user IDs when calling `GetDashboardOverview` for request-derived metrics.
- Add `user_ids` to alert event dimensions and human-readable descriptions.
- Keep account and group availability metrics unchanged unless they are already scoped by group/platform.

## Frontend Design

Ops dashboard state adds `userIds: number[]`.

The header receives `userIds` and emits `update:userIds`. It loads customer options via the existing admin user APIs. Search should debounce remote calls and use the existing admin user list/search behavior where possible. The selector must support multiple checked users and clear all.

API params include `user_ids` as a comma-separated string when at least one user is selected. Request details modal receives the same selected users; if the existing modal supports only one `user_id`, it should be extended to pass `user_ids` rather than silently dropping multi-select.

Alert rule types keep `filters?: Record<string, any>` and the editor writes `filters.user_ids = number[]`. Validation should remove `user_ids` when empty.

## Error Handling

- Invalid `user_ids` query values return `400` for backend API calls.
- Frontend URL parsing ignores invalid route values locally to keep page load resilient.
- If user option loading fails, dashboard data still works with IDs already in the URL, and the selector shows an empty option list with normal error toast behavior.
- Alert rules with invalid stored `filters.user_ids` are treated as unscoped rather than breaking evaluator cycles.

## Testing

Backend tests:

- Dashboard handler parses multiple `user_ids`.
- Repository query builders include user filters for usage and error queries.
- Alert evaluator applies `filters.user_ids` for request-derived metrics.
- Snapshot cache key distinguishes different user sets.

Frontend tests:

- Ops dashboard serializes and parses `user_ids`.
- Header emits multi-select changes.
- Alert rule editor saves and clears `filters.user_ids`.

Manual verification:

- Select two users and confirm dashboard network requests include `user_ids`.
- Clear selection and confirm requests omit `user_ids`.
- Create an alert rule scoped to selected users and confirm saved filters contain `user_ids`.
