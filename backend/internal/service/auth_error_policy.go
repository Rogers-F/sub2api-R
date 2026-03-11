package service

import (
	"net/http"
	"strings"
)

var oauthExpired401Markers = []string{
	"oauth token has expired",
	"token has expired",
	"access token expired",
	"expired token",
	"token expired",
	"session expired",
}

func isOAuthTokenExpired401(account *Account, statusCode int, responseBody []byte) bool {
	if account == nil || account.Type != AccountTypeOAuth || statusCode != http.StatusUnauthorized {
		return false
	}

	msg := strings.TrimSpace(extractUpstreamErrorMessage(responseBody))
	if msg == "" {
		msg = strings.TrimSpace(string(responseBody))
	}
	return isOAuth401ExpiredMessage(msg)
}

func isOAuth401ExpiredMessage(msg string) bool {
	normalized := strings.ToLower(strings.TrimSpace(sanitizeUpstreamErrorMessage(msg)))
	if normalized == "" {
		return false
	}

	for _, marker := range oauthExpired401Markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}

	return strings.Contains(normalized, "expired") &&
		(strings.Contains(normalized, "token") || strings.Contains(normalized, "session"))
}
