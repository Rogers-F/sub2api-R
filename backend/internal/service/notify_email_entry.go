package service

import (
	"encoding/json"
	"strings"
)

// NotifyEmailEntry represents a notification email with enable/disable and verification state.
type NotifyEmailEntry struct {
	Email    string `json:"email"`
	Disabled bool   `json:"disabled"`
	Verified bool   `json:"verified"`
}

// ParseNotifyEmails parses either the legacy []string format or the new []NotifyEmailEntry format.
func ParseNotifyEmails(raw string) []NotifyEmailEntry {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}

	var entries []NotifyEmailEntry
	if err := json.Unmarshal([]byte(raw), &entries); err == nil && len(entries) > 0 && !isOldStringArrayFormat(raw) {
		return entries
	}

	var emails []string
	if err := json.Unmarshal([]byte(raw), &emails); err == nil {
		result := make([]NotifyEmailEntry, 0, len(emails))
		for _, email := range emails {
			email = strings.TrimSpace(email)
			if email == "" {
				continue
			}
			result = append(result, NotifyEmailEntry{
				Email:    email,
				Disabled: false,
				Verified: false,
			})
		}
		return result
	}

	return nil
}

func isOldStringArrayFormat(raw string) bool {
	var arr []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &arr); err != nil || len(arr) == 0 {
		return false
	}
	first := strings.TrimSpace(string(arr[0]))
	return len(first) > 0 && first[0] == '"'
}

// MarshalNotifyEmails serializes entries back to JSON.
func MarshalNotifyEmails(entries []NotifyEmailEntry) string {
	if len(entries) == 0 {
		return "[]"
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return "[]"
	}
	return string(data)
}
