package service

import (
	"crypto/rand"
	"time"
)

type RedeemCode struct {
	ID        int64
	Code      string
	Type      string
	Value     float64
	Status    string
	UsedBy    *int64
	UsedAt    *time.Time
	Notes     string
	CreatedAt time.Time

	GroupID      *int64
	ValidityDays int

	User  *User
	Group *Group
}

func (r *RedeemCode) IsUsed() bool {
	return r.Status == StatusUsed
}

func (r *RedeemCode) CanUse() bool {
	return r.Status == StatusUnused
}

// codeAlphabet 兑换码字符集（去除易混淆字符 I, L, O）
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ"
const codeLength = 8

func GenerateRedeemCode() (string, error) {
	b := make([]byte, codeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	result := make([]byte, codeLength)
	for i := 0; i < codeLength; i++ {
		result[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(result), nil
}
