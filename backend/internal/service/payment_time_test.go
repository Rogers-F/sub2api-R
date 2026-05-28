package service

import (
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

func TestGenerateOutTradeNoAtUsesBeijingDate(t *testing.T) {
	utcInstant := time.Date(2026, 1, 1, 16, 30, 0, 0, time.UTC)

	got := generateOutTradeNoAt(utcInstant)

	if !strings.HasPrefix(got, orderIDPrefix+"20260102") {
		t.Fatalf("generateOutTradeNoAt(%s) = %s, want Beijing date prefix %s", utcInstant.Format(time.RFC3339), got, orderIDPrefix+"20260102")
	}
}

func TestPaymentStartOfDayUsesBeijingDay(t *testing.T) {
	utcInstant := time.Date(2026, 1, 1, 16, 30, 0, 0, time.UTC)

	got := psStartOfDayBeijing(utcInstant)
	want := time.Date(2026, 1, 2, 0, 0, 0, 0, timezone.BeijingLocation())

	if !got.Equal(want) {
		t.Fatalf("psStartOfDayBeijing(%s) = %s, want %s", utcInstant.Format(time.RFC3339), got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
	if got.Location().String() != timezone.BeijingTimezone {
		t.Fatalf("psStartOfDayBeijing location = %s, want %s", got.Location().String(), timezone.BeijingTimezone)
	}
}
