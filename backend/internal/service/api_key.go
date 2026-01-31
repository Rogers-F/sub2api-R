package service

import "time"

type APIKey struct {
	ID          int64
	UserID      int64
	Key         string
	Name        string
	GroupID     *int64
	Status      string
	IPWhitelist []string
	IPBlacklist []string
	QuotaUSD    *float64 // Usage quota in USD, nil means unlimited
	UsedUSD     float64  // Accumulated actual cost
	CreatedAt   time.Time
	UpdatedAt   time.Time
	User        *User
	Group       *Group
}

func (k *APIKey) IsActive() bool {
	return k.Status == StatusActive
}

// HasQuota returns true if this API key has a quota limit set
func (k *APIKey) HasQuota() bool {
	return k.QuotaUSD != nil && *k.QuotaUSD > 0
}

// IsQuotaExceeded returns true if usage has reached or exceeded the quota
func (k *APIKey) IsQuotaExceeded() bool {
	return k.HasQuota() && k.UsedUSD >= *k.QuotaUSD
}
