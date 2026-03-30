package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	PaygOrderStatusPending = "PENDING"
	PaygOrderStatusPaid    = "PAID"
	PaygOrderStatusClosed  = "CLOSED"

	PaygPaywayAlipay = "1"
	PaygPaywayWeChat = "3"
)

var (
	ErrPaygDisabled              = infraerrors.Forbidden("PAYG_DISABLED", "payg wallet is disabled")
	ErrPaygInvalidAmount         = infraerrors.BadRequest("PAYG_INVALID_AMOUNT", "invalid payg amount")
	ErrPaygOrderNotFound         = infraerrors.NotFound("PAYG_ORDER_NOT_FOUND", "payg order not found")
	ErrPaygAmountMismatch        = infraerrors.Conflict("PAYG_AMOUNT_MISMATCH", "payg order amount mismatch")
	ErrPaygProviderNotConfigured = infraerrors.BadRequest("PAYG_PROVIDER_NOT_CONFIGURED", "payg provider is not configured")
	ErrPaygProviderRejected      = infraerrors.ServiceUnavailable("PAYG_PROVIDER_REJECTED", "payg provider rejected request")
	ErrPaygCallbackUnauthorized  = infraerrors.Forbidden("PAYG_CALLBACK_UNAUTHORIZED", "invalid payg callback token")
	ErrPaygCallbackUnconfigured  = infraerrors.ServiceUnavailable("PAYG_CALLBACK_UNCONFIGURED", "payg callback token is not configured")
)

type PaygSettings struct {
	Enabled            bool
	ExchangeRate       float64
	FixedAmountOptions []float64
	TerminalSN         string
	TerminalKey        string
}

type PaygOrder struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	ClientSN     string     `json:"client_sn"`
	SN           string     `json:"sn"`
	AmountYuan   float64    `json:"amount_yuan"`
	AmountCent   int64      `json:"amount_cent"`
	CreditAmount float64    `json:"credit_amount"`
	Payway       string     `json:"payway"`
	PaywayName   string     `json:"payway_name"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
}

type PaygWallet struct {
	Enabled             bool         `json:"enabled"`
	Balance             float64      `json:"balance"`
	ExchangeRate        float64      `json:"exchange_rate"`
	FixedAmountOptions  []float64    `json:"fixed_amount_options"`
	TotalPaidAmount     float64      `json:"total_paid_amount"`
	TotalCreditedAmount float64      `json:"total_credited_amount"`
	TotalConsumption    float64      `json:"total_consumption"`
	Orders              []*PaygOrder `json:"orders"`
}

type PaygAdminUserSummary struct {
	UserID              int64   `json:"user_id"`
	Email               string  `json:"email"`
	OrderCount          int     `json:"order_count"`
	TotalPaidAmount     float64 `json:"total_paid_amount"`
	TotalCreditedAmount float64 `json:"total_credited_amount"`
}

type PaygAdminOrderItem struct {
	PaygOrder
	Email string `json:"email"`
}

type PaygAdminWallet struct {
	Enabled             bool                    `json:"enabled"`
	TotalOrders         int                     `json:"total_orders"`
	PaidOrders          int                     `json:"paid_orders"`
	PendingOrders       int                     `json:"pending_orders"`
	TotalPaidAmount     float64                 `json:"total_paid_amount"`
	TotalCreditedAmount float64                 `json:"total_credited_amount"`
	Users               []*PaygAdminUserSummary `json:"users"`
	Orders              []*PaygAdminOrderItem   `json:"orders"`
}

type PaygUserSummary struct {
	TotalPaidAmount     float64
	TotalCreditedAmount float64
	Orders              []*PaygOrder
}

type PaygAdminSummary struct {
	TotalOrders         int
	PaidOrders          int
	PendingOrders       int
	TotalPaidAmount     float64
	TotalCreditedAmount float64
	Users               []*PaygAdminUserSummary
	Orders              []*PaygAdminOrderItem
}

type PaygProviderOrderStatus struct {
	ClientSN        string
	SN              string
	Status          string
	Payway          string
	PaywayName      string
	TotalAmountCent int64
	PaidAt          *time.Time
}

type PaygOrderRepository interface {
	Create(ctx context.Context, order *PaygOrder) error
	GetByIDForUser(ctx context.Context, orderID, userID int64) (*PaygOrder, error)
	GetByIdentifiers(ctx context.Context, sn, clientSN string) (*PaygOrder, error)
	GetForUpdateByIdentifiers(ctx context.Context, sn, clientSN string) (*PaygOrder, error)
	HasPendingOrders(ctx context.Context) (bool, error)
	UpdateProviderState(ctx context.Context, order *PaygOrder) error
	MarkPaid(ctx context.Context, order *PaygOrder) error
	GetUserSummary(ctx context.Context, userID int64, orderLimit int) (*PaygUserSummary, error)
	GetAdminSummary(ctx context.Context, userLimit, orderLimit int) (*PaygAdminSummary, error)
}
