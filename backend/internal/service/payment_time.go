package service

import (
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

func paymentBeijingNow() time.Time {
	return timezone.BeijingNow()
}

func paymentInBeijing(t time.Time) time.Time {
	return timezone.InBeijing(t)
}

func paymentTimePtrInBeijing(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	converted := paymentInBeijing(*t)
	return &converted
}

func normalizePaymentOrderTimes(o *dbent.PaymentOrder) *dbent.PaymentOrder {
	if o == nil {
		return nil
	}
	o.CreatedAt = paymentInBeijing(o.CreatedAt)
	o.UpdatedAt = paymentInBeijing(o.UpdatedAt)
	o.ExpiresAt = paymentInBeijing(o.ExpiresAt)
	o.PaidAt = paymentTimePtrInBeijing(o.PaidAt)
	o.CompletedAt = paymentTimePtrInBeijing(o.CompletedAt)
	o.FailedAt = paymentTimePtrInBeijing(o.FailedAt)
	o.RefundAt = paymentTimePtrInBeijing(o.RefundAt)
	o.RefundRequestedAt = paymentTimePtrInBeijing(o.RefundRequestedAt)
	return o
}

func normalizePaymentOrdersTimes(orders []*dbent.PaymentOrder) []*dbent.PaymentOrder {
	for _, order := range orders {
		normalizePaymentOrderTimes(order)
	}
	return orders
}

func normalizePaymentAuditLogTimes(log *dbent.PaymentAuditLog) *dbent.PaymentAuditLog {
	if log == nil {
		return nil
	}
	log.CreatedAt = paymentInBeijing(log.CreatedAt)
	return log
}

func normalizePaymentAuditLogsTimes(logs []*dbent.PaymentAuditLog) []*dbent.PaymentAuditLog {
	for _, log := range logs {
		normalizePaymentAuditLogTimes(log)
	}
	return logs
}

func psStartOfDayBeijing(t time.Time) time.Time {
	return timezone.StartOfBeijingDay(t)
}
