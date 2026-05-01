# Enterprise Ops RPM Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show dashboard-consistent enterprise RPM, enterprise 5-minute error rate, enterprise filtering in Ops, and account scheduling toggles inside enterprise detail.

**Architecture:** Enterprise list metrics are computed server-side in one batch for the current page. Ops dashboard filters carry `enterprise_id` through API, service, and repository query builders. Account scheduling uses the existing account schedulable endpoint.

**Tech Stack:** Go backend with PostgreSQL queries, Gin handlers, Vue 3 frontend, TypeScript admin API clients.

---

### Task 1: Enterprise Metrics

**Files:**
- Modify: `backend/internal/service/enterprise.go`
- Modify: `backend/internal/repository/enterprise_repo.go`
- Modify: `backend/internal/handler/dto/types.go`
- Modify: `backend/internal/handler/dto/mappers.go`

- [ ] Add `RPM int64` and `ErrorRate5m float64` to the service and DTO enterprise structs.
- [ ] In `ListWithFilters` and `GetByID`, batch-load metrics for enterprise IDs.
- [ ] Match admin dashboard RPM exactly: last 5 minutes, `usage_logs` only, `COUNT(*) / 5`.
- [ ] Compute 5-minute error rate with Ops SLA semantics: `error_count_sla / (success_count + error_count_sla)`.

### Task 2: Ops Enterprise Filter

**Files:**
- Modify: `backend/internal/service/ops_dashboard_models.go`
- Modify: `backend/internal/service/ops_models.go`
- Modify: `backend/internal/service/ops_request_details.go`
- Modify: `backend/internal/handler/admin/ops_dashboard_handler.go`
- Modify: `backend/internal/handler/admin/ops_handler.go`
- Modify: `backend/internal/repository/ops_repo_dashboard.go`
- Modify: `backend/internal/repository/ops_repo.go`
- Modify: `backend/internal/repository/ops_repo_request_details.go`

- [ ] Add `EnterpriseID *int64` to dashboard, error-log, and request-detail filters.
- [ ] Parse `enterprise_id` from Ops handlers.
- [ ] Add enterprise filtering to `usage_logs` queries via `accounts.enterprise_id`.
- [ ] Add enterprise filtering to `ops_error_logs` queries with an account `EXISTS` filter.
- [ ] Force raw querying whenever `enterprise_id` is present because hourly pre-aggregation has no enterprise dimension.

### Task 3: Frontend

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/admin/ops.ts`
- Modify: `frontend/src/views/admin/EnterprisesView.vue`
- Modify: `frontend/src/views/admin/EnterpriseDetailView.vue`
- Modify: `frontend/src/views/admin/ops/OpsDashboard.vue`
- Modify: `frontend/src/views/admin/ops/components/OpsDashboardHeader.vue`
- Modify: `frontend/src/views/admin/ops/components/OpsRequestDetailsModal.vue`
- Modify: `frontend/src/views/admin/ops/components/OpsErrorDetailsModal.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] Show `RPM` and `5分钟错误率` on enterprise list/detail.
- [ ] Add enterprise filter to Ops header and route query.
- [ ] Pass `enterprise_id` to Ops dashboard APIs and detail modals.
- [ ] Add per-account scheduling switch in enterprise account table using `adminAPI.accounts.setSchedulable`.

### Task 4: Verification

- [ ] Run focused Go tests or `go test ./...` if feasible.
- [ ] Run frontend typecheck/build.
- [ ] Run `git diff --check`.
