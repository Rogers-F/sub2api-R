# Enterprise Management Design

## Goal

Add an independent enterprise management layer for admin users. Enterprises group upstream accounts for operational management only. They do not affect request scheduling, billing, group routing, user permissions, or account status.

Admins can choose an enterprise when creating or editing an account. The admin sidebar includes an Enterprise Management entry. Clicking an enterprise opens a detail page where admins can manage the accounts assigned to that enterprise, create new accounts under it, and bulk move accounts in or out.

## Data Model

Add a new `enterprises` table:

- `id BIGSERIAL PRIMARY KEY`
- `name VARCHAR(100) NOT NULL`
- `notes TEXT`
- `status VARCHAR(20) NOT NULL DEFAULT 'active'`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `deleted_at TIMESTAMPTZ`

Add `enterprise_id BIGINT NULL` to `accounts`, with a foreign key to `enterprises(id)` using `ON DELETE SET NULL`.

The relationship is one-to-many:

- One enterprise can own many accounts.
- One account can belong to at most one enterprise.
- An account can have no enterprise.

Enterprise names must be unique among non-deleted enterprises. Soft-deleted enterprise names can be reused.

## Backend API

Add admin-only enterprise endpoints:

- `GET /api/v1/admin/enterprises`
  - Paged list.
  - Filters: `search`, `status`.
  - Response includes account count.
- `POST /api/v1/admin/enterprises`
  - Creates an enterprise.
  - Fields: `name`, `notes`, `status`.
- `GET /api/v1/admin/enterprises/:id`
  - Returns enterprise detail and account count.
- `PUT /api/v1/admin/enterprises/:id`
  - Updates `name`, `notes`, `status`.
- `DELETE /api/v1/admin/enterprises/:id`
  - Soft-deletes the enterprise.
  - Sets all linked `accounts.enterprise_id` values to `NULL`.
- `GET /api/v1/admin/enterprises/:id/accounts`
  - Lists accounts assigned to the enterprise.
  - Reuses existing account pagination and filters where practical.
- `POST /api/v1/admin/enterprises/:id/accounts`
  - Bulk moves accounts into the enterprise.
  - Body: `{ "account_ids": [1, 2, 3] }`.
- `DELETE /api/v1/admin/enterprises/:id/accounts`
  - Bulk removes accounts from the enterprise.
  - Body: `{ "account_ids": [1, 2, 3] }`.

Extend existing account APIs:

- `GET /api/v1/admin/accounts?enterprise=123`
- `GET /api/v1/admin/accounts?enterprise=unassigned`
- `POST /api/v1/admin/accounts` accepts `enterprise_id`.
- `PUT /api/v1/admin/accounts/:id` accepts `enterprise_id`.
- Bulk account update accepts `enterprise_id` for bulk move and `enterprise_id: null` for bulk clear.

The account response includes:

- `enterprise_id`
- `enterprise` shallow object when preloaded: `id`, `name`, `status`

## Business Rules

Enterprise status values are `active` and `disabled`.

Creating or editing an account:

- `enterprise_id` may be omitted or null.
- A non-null `enterprise_id` must reference an active, non-deleted enterprise.
- Existing accounts assigned to a disabled enterprise keep their assignment.

Bulk moving accounts:

- Moving into an enterprise requires the target enterprise to be active.
- Moving into an enterprise replaces any previous enterprise assignment because the relation is one-to-many.
- Moving out sets `enterprise_id` to null.

Deleting an enterprise:

- Soft-delete the enterprise.
- Set all assigned accounts to no enterprise.
- Do not delete accounts.
- Do not change account status, schedulable state, groups, credentials, proxy, billing multiplier, or usage logs.

Scheduling and billing:

- Enterprise does not participate in scheduler selection.
- Enterprise does not change account grouping, user allowed groups, API key group selection, usage costs, or subscription logic.

## Frontend

Add sidebar item:

- Label: `企业管理` / `Enterprise Management`
- Route: `/admin/enterprises`
- Admin only.

Add routes:

- `/admin/enterprises`
- `/admin/enterprises/:id`

Enterprise list page:

- Table columns: name, status, notes, account count, created at, actions.
- Actions: create, edit, enable, disable, delete.
- Click name to open detail page.

Enterprise detail page:

- Header with enterprise name, status, notes, and account count.
- Account table filtered to this enterprise.
- Create account button opens the existing create account modal with `enterprise_id` preselected.
- Existing account edit opens the existing edit account modal.
- Bulk move in opens an account picker that can search accounts from all enterprises and unassigned accounts.
- Bulk move out clears selected accounts from the current enterprise.

Admin account page changes:

- Add enterprise filter with active enterprises and `无企业` / `Unassigned`.
- Add enterprise column with enterprise name or `无企业`.
- Add enterprise selector in create/edit account modals.
- Bulk edit supports setting or clearing enterprise.

## Migration Strategy

Add a new numbered SQL migration:

- Create `enterprises`.
- Add `accounts.enterprise_id`.
- Add indexes for `enterprises.status`, `enterprises.deleted_at`, `accounts.enterprise_id`.
- Add a partial unique index on `lower(name)` for non-deleted enterprises.

Add Ent schema changes for Enterprise and Account enterprise edge.

Existing accounts start with `enterprise_id = NULL`.

## Testing

Backend tests:

- Enterprise create/list/get/update/delete.
- Duplicate active enterprise names are rejected.
- Soft-deleted enterprise names can be reused.
- Deleting an enterprise clears linked account enterprise IDs.
- Account list filters by enterprise ID and unassigned.
- Account create/update validates active enterprise ID.
- Disabled enterprises cannot receive new or moved accounts.
- Bulk move in and move out update only enterprise assignment.

Frontend tests:

- Sidebar renders enterprise management for admins.
- Enterprise routes are registered as admin routes.
- Account create/edit payload includes `enterprise_id`.
- Account list filter sends `enterprise` query.
- Enterprise detail bulk move in/out calls the expected APIs.

Regression checks:

- Existing account create/edit without enterprise still works.
- Existing group assignment behavior is unchanged.
- Existing account scheduling and usage views do not require enterprise data.

## Rollout

This is a normal additive migration. It is safe for existing deployments because new enterprise fields are nullable and existing account behavior remains unchanged until admins assign enterprises.
