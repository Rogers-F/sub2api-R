package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

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
	metrics, err := r.enterpriseMetrics5m(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	for i := range out {
		if metric, ok := metrics[out[i].ID]; ok {
			out[i].RPM = metric.rpm
			out[i].ErrorRate5m = metric.errorRate5m
		}
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
	metrics, err := r.enterpriseMetrics5m(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if metric, ok := metrics[id]; ok {
		out.RPM = metric.rpm
		out.ErrorRate5m = metric.errorRate5m
	}
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

type enterpriseMetric5m struct {
	successCount int64
	errorCount   int64
	rpm          int64
	errorRate5m  float64
}

func (r *enterpriseRepository) enterpriseMetrics5m(ctx context.Context, ids []int64) (map[int64]enterpriseMetric5m, error) {
	metrics := make(map[int64]enterpriseMetric5m, len(ids))
	ids = normalizePositiveIDs(ids)
	if len(ids) == 0 || r.sql == nil {
		return metrics, nil
	}

	windowStart := time.Now().Add(-5 * time.Minute)

	rows, err := r.sql.QueryContext(ctx, `
		SELECT a.enterprise_id, COUNT(*)::bigint
		FROM usage_logs ul
		JOIN accounts a ON a.id = ul.account_id
		WHERE ul.created_at >= $1
			AND a.enterprise_id = ANY($2)
			AND a.deleted_at IS NULL
		GROUP BY a.enterprise_id
	`, windowStart, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var enterpriseID int64
		var successCount int64
		if err := rows.Scan(&enterpriseID, &successCount); err != nil {
			_ = rows.Close()
			return nil, err
		}
		metric := metrics[enterpriseID]
		metric.successCount = successCount
		metric.rpm = successCount / 5
		metrics[enterpriseID] = metric
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = r.sql.QueryContext(ctx, `
		SELECT a.enterprise_id, COUNT(*)::bigint
		FROM ops_error_logs e
		JOIN accounts a ON a.id = e.account_id
		WHERE e.created_at >= $1
			AND COALESCE(e.is_count_tokens, FALSE) = FALSE
			AND COALESCE(e.status_code, 0) >= 400
			AND COALESCE(e.is_business_limited, FALSE) = FALSE
			AND a.enterprise_id = ANY($2)
			AND a.deleted_at IS NULL
		GROUP BY a.enterprise_id
	`, windowStart, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var enterpriseID int64
		var errorCount int64
		if err := rows.Scan(&enterpriseID, &errorCount); err != nil {
			_ = rows.Close()
			return nil, err
		}
		metric := metrics[enterpriseID]
		metric.errorCount = errorCount
		denominator := metric.successCount + metric.errorCount
		if denominator > 0 {
			metric.errorRate5m = roundEnterpriseMetric4DP(float64(metric.errorCount) / float64(denominator))
		}
		metrics[enterpriseID] = metric
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return metrics, nil
}

func roundEnterpriseMetric4DP(v float64) float64 {
	return math.Round(v*10000) / 10000
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
