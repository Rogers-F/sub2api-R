package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type paygOrderRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewPaygOrderRepository(client *dbent.Client, sqlDB *sql.DB) service.PaygOrderRepository {
	return &paygOrderRepository{
		client: client,
		sql:    sqlDB,
	}
}

func (r *paygOrderRepository) sqlQueryerFromContext(ctx context.Context) sqlQueryer {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.sql
}

func (r *paygOrderRepository) sqlExecutorFromContext(ctx context.Context) sqlExecutor {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.sql
}

func (r *paygOrderRepository) Create(ctx context.Context, order *service.PaygOrder) error {
	query := `
		INSERT INTO payg_orders (
			user_id, client_sn, sn, amount_yuan, amount_cent, credit_amount, payway, payway_name, status, created_at, updated_at
		) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	if err := scanSingleRow(ctx, r.sqlQueryerFromContext(ctx), query, []any{
		order.UserID,
		order.ClientSN,
		order.SN,
		order.AmountYuan,
		order.AmountCent,
		order.CreditAmount,
		order.Payway,
		order.PaywayName,
		order.Status,
	}, &order.ID, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return fmt.Errorf("create payg order: %w", err)
	}
	return nil
}

func (r *paygOrderRepository) GetByIDForUser(ctx context.Context, orderID, userID int64) (*service.PaygOrder, error) {
	query := `
		SELECT id, user_id, client_sn, COALESCE(sn, ''), amount_yuan, amount_cent, credit_amount,
		       COALESCE(payway, ''), COALESCE(payway_name, ''), status, created_at, updated_at, paid_at
		FROM payg_orders
		WHERE id = $1 AND user_id = $2
	`
	order, err := r.getOne(ctx, query, []any{orderID, userID})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrPaygOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

func (r *paygOrderRepository) GetByIdentifiers(ctx context.Context, sn, clientSN string) (*service.PaygOrder, error) {
	return r.getByIdentifiers(ctx, sn, clientSN, false)
}

func (r *paygOrderRepository) GetForUpdateByIdentifiers(ctx context.Context, sn, clientSN string) (*service.PaygOrder, error) {
	return r.getByIdentifiers(ctx, sn, clientSN, true)
}

func (r *paygOrderRepository) getByIdentifiers(ctx context.Context, sn, clientSN string, forUpdate bool) (*service.PaygOrder, error) {
	query := `
		SELECT id, user_id, client_sn, COALESCE(sn, ''), amount_yuan, amount_cent, credit_amount,
		       COALESCE(payway, ''), COALESCE(payway_name, ''), status, created_at, updated_at, paid_at
		FROM payg_orders
		WHERE (($1 <> '' AND sn = $1) OR ($2 <> '' AND client_sn = $2))
		ORDER BY id DESC
		LIMIT 1
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	order, err := r.getOne(ctx, query, []any{sn, clientSN})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrPaygOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

func (r *paygOrderRepository) UpdateProviderState(ctx context.Context, order *service.PaygOrder) error {
	query := `
		UPDATE payg_orders
		SET sn = NULLIF($2, ''),
		    payway = $3,
		    payway_name = $4,
		    status = $5,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	if err := scanSingleRow(ctx, r.sqlQueryerFromContext(ctx), query, []any{
		order.ID,
		order.SN,
		order.Payway,
		order.PaywayName,
		order.Status,
	}, &order.UpdatedAt); err != nil {
		return fmt.Errorf("update payg provider state: %w", err)
	}
	return nil
}

func (r *paygOrderRepository) MarkPaid(ctx context.Context, order *service.PaygOrder) error {
	query := `
		UPDATE payg_orders
		SET sn = NULLIF($2, ''),
		    payway = $3,
		    payway_name = $4,
		    status = $5,
		    paid_at = $6,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	if err := scanSingleRow(ctx, r.sqlQueryerFromContext(ctx), query, []any{
		order.ID,
		order.SN,
		order.Payway,
		order.PaywayName,
		service.PaygOrderStatusPaid,
		order.PaidAt,
	}, &order.UpdatedAt); err != nil {
		return fmt.Errorf("mark payg order paid: %w", err)
	}
	order.Status = service.PaygOrderStatusPaid
	return nil
}

func (r *paygOrderRepository) GetUserSummary(ctx context.Context, userID int64, orderLimit int) (*service.PaygUserSummary, error) {
	summary := &service.PaygUserSummary{}
	if err := scanSingleRow(ctx, r.sqlQueryerFromContext(ctx), `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'PAID' THEN amount_yuan ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'PAID' THEN credit_amount ELSE 0 END), 0)
		FROM payg_orders
		WHERE user_id = $1
	`, []any{userID}, &summary.TotalPaidAmount, &summary.TotalCreditedAmount); err != nil {
		return nil, fmt.Errorf("get payg user summary: %w", err)
	}

	orders, err := r.listOrders(ctx, `
		SELECT id, user_id, client_sn, COALESCE(sn, ''), amount_yuan, amount_cent, credit_amount,
		       COALESCE(payway, ''), COALESCE(payway_name, ''), status, created_at, updated_at, paid_at
		FROM payg_orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, []any{userID, orderLimit})
	if err != nil {
		return nil, fmt.Errorf("list payg user orders: %w", err)
	}
	summary.Orders = orders
	return summary, nil
}

func (r *paygOrderRepository) GetAdminSummary(ctx context.Context, userLimit, orderLimit int) (*service.PaygAdminSummary, error) {
	summary := &service.PaygAdminSummary{}
	if err := scanSingleRow(ctx, r.sqlQueryerFromContext(ctx), `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'PAID' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'PENDING' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'PAID' THEN amount_yuan ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'PAID' THEN credit_amount ELSE 0 END), 0)
		FROM payg_orders
	`, nil, &summary.TotalOrders, &summary.PaidOrders, &summary.PendingOrders, &summary.TotalPaidAmount, &summary.TotalCreditedAmount); err != nil {
		return nil, fmt.Errorf("get payg admin totals: %w", err)
	}

	userRows, err := r.sqlQueryerFromContext(ctx).QueryContext(ctx, `
		SELECT o.user_id,
		       COALESCE(u.email, 'unknown'),
		       COUNT(*),
		       COALESCE(SUM(CASE WHEN o.status = 'PAID' THEN o.amount_yuan ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN o.status = 'PAID' THEN o.credit_amount ELSE 0 END), 0)
		FROM payg_orders o
		LEFT JOIN users u ON u.id = o.user_id
		GROUP BY o.user_id, u.email
		ORDER BY COALESCE(SUM(CASE WHEN o.status = 'PAID' THEN o.credit_amount ELSE 0 END), 0) DESC, o.user_id DESC
		LIMIT $1
	`, userLimit)
	if err != nil {
		return nil, fmt.Errorf("list payg admin users: %w", err)
	}
	defer func() { _ = userRows.Close() }()

	summary.Users = make([]*service.PaygAdminUserSummary, 0, userLimit)
	for userRows.Next() {
		item := &service.PaygAdminUserSummary{}
		if err := userRows.Scan(&item.UserID, &item.Email, &item.OrderCount, &item.TotalPaidAmount, &item.TotalCreditedAmount); err != nil {
			return nil, fmt.Errorf("scan payg admin user summary: %w", err)
		}
		summary.Users = append(summary.Users, item)
	}
	if err := userRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payg admin users: %w", err)
	}

	orderRows, err := r.sqlQueryerFromContext(ctx).QueryContext(ctx, `
		SELECT o.id, o.user_id, o.client_sn, COALESCE(o.sn, ''), o.amount_yuan, o.amount_cent, o.credit_amount,
		       COALESCE(o.payway, ''), COALESCE(o.payway_name, ''), o.status, o.created_at, o.updated_at, o.paid_at,
		       COALESCE(u.email, 'unknown')
		FROM payg_orders o
		LEFT JOIN users u ON u.id = o.user_id
		ORDER BY o.created_at DESC
		LIMIT $1
	`, orderLimit)
	if err != nil {
		return nil, fmt.Errorf("list payg admin orders: %w", err)
	}
	defer func() { _ = orderRows.Close() }()

	summary.Orders = make([]*service.PaygAdminOrderItem, 0, orderLimit)
	for orderRows.Next() {
		item := &service.PaygAdminOrderItem{}
		paidAt, err := scanPaygAdminOrderColumns(orderRows, item)
		if err != nil {
			return nil, err
		}
		item.PaidAt = paidAt
		summary.Orders = append(summary.Orders, item)
	}
	if err := orderRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payg admin orders: %w", err)
	}

	return summary, nil
}

func (r *paygOrderRepository) getOne(ctx context.Context, query string, args []any) (*service.PaygOrder, error) {
	rows, err := r.sqlQueryerFromContext(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}

	order := &service.PaygOrder{}
	paidAt, err := scanPaygOrderColumns(
		rows,
		&order.ID,
		&order.UserID,
		&order.ClientSN,
		&order.SN,
		&order.AmountYuan,
		&order.AmountCent,
		&order.CreditAmount,
		&order.Payway,
		&order.PaywayName,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	order.PaidAt = paidAt
	return order, rows.Err()
}

func (r *paygOrderRepository) listOrders(ctx context.Context, query string, args []any) ([]*service.PaygOrder, error) {
	rows, err := r.sqlQueryerFromContext(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	orders := make([]*service.PaygOrder, 0)
	for rows.Next() {
		order := &service.PaygOrder{}
		paidAt, err := scanPaygOrderColumns(
			rows,
			&order.ID,
			&order.UserID,
			&order.ClientSN,
			&order.SN,
			&order.AmountYuan,
			&order.AmountCent,
			&order.CreditAmount,
			&order.Payway,
			&order.PaywayName,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		order.PaidAt = paidAt
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

type paygRowScanner interface {
	Scan(dest ...any) error
}

func scanPaygOrderColumns(scanner paygRowScanner, dest ...any) (*time.Time, error) {
	var paidAt sql.NullTime
	scanDest := append(dest, &paidAt)
	if err := scanner.Scan(scanDest...); err != nil {
		return nil, fmt.Errorf("scan payg order: %w", err)
	}
	if paidAt.Valid {
		paidTime := paidAt.Time
		return &paidTime, nil
	}
	return nil, nil
}

func scanPaygAdminOrderColumns(scanner paygRowScanner, item *service.PaygAdminOrderItem) (*time.Time, error) {
	var paidAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.ClientSN,
		&item.SN,
		&item.AmountYuan,
		&item.AmountCent,
		&item.CreditAmount,
		&item.Payway,
		&item.PaywayName,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
		&paidAt,
		&item.Email,
	); err != nil {
		return nil, fmt.Errorf("scan payg admin order: %w", err)
	}
	if paidAt.Valid {
		paidTime := paidAt.Time
		return &paidTime, nil
	}
	return nil, nil
}
