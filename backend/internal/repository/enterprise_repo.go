package repository

import (
	"context"
	"database/sql"
	"errors"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbenterprise "github.com/Wei-Shaw/sub2api/ent/enterprise"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type enterpriseRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewEnterpriseRepository(client *dbent.Client, sqlDB *sql.DB) service.EnterpriseRepository {
	return &enterpriseRepository{client: client, sql: sqlDB}
}

func (r *enterpriseRepository) Create(ctx context.Context, enterprise *service.Enterprise) error {
	if enterprise == nil {
		return nil
	}
	created, err := r.client.Enterprise.Create().
		SetName(enterprise.Name).
		SetNillableNotes(enterprise.Notes).
		SetStatus(enterprise.Status).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrEnterpriseNotFound, service.ErrEnterpriseNameTaken)
	}
	*enterprise = *enterpriseEntityToService(created)
	return nil
}

func (r *enterpriseRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.EnterpriseListFilters) ([]service.Enterprise, *pagination.PaginationResult, error) {
	q := r.client.Enterprise.Query()
	if filters.Search != "" {
		q = q.Where(dbenterprise.Or(
			dbenterprise.NameContainsFold(filters.Search),
			dbenterprise.NotesContainsFold(filters.Search),
		))
	}
	if filters.Status != "" {
		q = q.Where(dbenterprise.StatusEQ(filters.Status))
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	entities, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(dbenterprise.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]service.Enterprise, 0, len(entities))
	ids := make([]int64, 0, len(entities))
	for _, entity := range entities {
		item := enterpriseEntityToService(entity)
		if item == nil {
			continue
		}
		out = append(out, *item)
		ids = append(ids, item.ID)
	}
	counts, err := r.countAccountsByEnterprise(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	for i := range out {
		out[i].AccountCount = counts[out[i].ID]
	}
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *enterpriseRepository) GetByID(ctx context.Context, id int64) (*service.Enterprise, error) {
	entity, err := r.client.Enterprise.Query().
		Where(dbenterprise.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrEnterpriseNotFound, nil)
	}
	out := enterpriseEntityToService(entity)
	counts, err := r.countAccountsByEnterprise(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	out.AccountCount = counts[id]
	return out, nil
}

func (r *enterpriseRepository) GetActiveByID(ctx context.Context, id int64) (*service.Enterprise, error) {
	enterprise, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if enterprise.Status != service.StatusActive {
		return nil, service.ErrEnterpriseNotActive
	}
	return enterprise, nil
}

func (r *enterpriseRepository) Update(ctx context.Context, enterprise *service.Enterprise) error {
	if enterprise == nil {
		return nil
	}
	builder := r.client.Enterprise.UpdateOneID(enterprise.ID).
		SetName(enterprise.Name).
		SetStatus(enterprise.Status)
	if enterprise.Notes != nil {
		builder.SetNotes(*enterprise.Notes)
	} else {
		builder.ClearNotes()
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrEnterpriseNotFound, service.ErrEnterpriseNameTaken)
	}
	*enterprise = *enterpriseEntityToService(updated)
	return nil
}

func (r *enterpriseRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return err
	}

	var txClient *dbent.Client
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		txClient = tx.Client()
	} else {
		txClient = r.client
	}

	if _, err := txClient.Account.Update().
		Where(dbaccount.EnterpriseIDEQ(id)).
		ClearEnterpriseID().
		Save(ctx); err != nil {
		return err
	}
	if err := txClient.Enterprise.DeleteOneID(id).Exec(ctx); err != nil {
		return translatePersistenceError(err, service.ErrEnterpriseNotFound, nil)
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (r *enterpriseRepository) AssignAccounts(ctx context.Context, enterpriseID int64, accountIDs []int64) (int64, error) {
	ids := normalizePositiveIDs(accountIDs)
	if len(ids) == 0 {
		return 0, nil
	}
	affected, err := r.client.Account.Update().
		Where(dbaccount.IDIn(ids...)).
		SetEnterpriseID(enterpriseID).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return int64(affected), nil
}

func (r *enterpriseRepository) UnassignAccounts(ctx context.Context, enterpriseID int64, accountIDs []int64) (int64, error) {
	ids := normalizePositiveIDs(accountIDs)
	if len(ids) == 0 {
		return 0, nil
	}
	affected, err := r.client.Account.Update().
		Where(
			dbaccount.IDIn(ids...),
			dbaccount.EnterpriseIDEQ(enterpriseID),
		).
		ClearEnterpriseID().
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return int64(affected), nil
}

func (r *enterpriseRepository) countAccountsByEnterprise(ctx context.Context, ids []int64) (map[int64]int64, error) {
	counts := make(map[int64]int64, len(ids))
	ids = normalizePositiveIDs(ids)
	if len(ids) == 0 || r.sql == nil {
		return counts, nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT enterprise_id, COUNT(*)
		FROM accounts
		WHERE enterprise_id = ANY($1)
			AND deleted_at IS NULL
		GROUP BY enterprise_id
	`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var enterpriseID int64
		var count int64
		if err := rows.Scan(&enterpriseID, &count); err != nil {
			return nil, err
		}
		counts[enterpriseID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func enterpriseEntityToService(entity *dbent.Enterprise) *service.Enterprise {
	if entity == nil {
		return nil
	}
	return &service.Enterprise{
		ID:        entity.ID,
		Name:      entity.Name,
		Notes:     entity.Notes,
		Status:    entity.Status,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}

func normalizePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
