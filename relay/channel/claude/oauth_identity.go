package claude

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// newMetadataFormatMinVersion is the minimum Claude Code version that emits the
// JSON-formatted metadata.user_id instead of the legacy concatenated string.
const newMetadataFormatMinVersion = "2.1.78"

var claudeCLIVersionRe = regexp.MustCompile(`(?i)^claude-cli/(\d+\.\d+\.\d+)`)

// defaultOAuthUAVersion is computed once from the (constant) OAuth fingerprint UA.
var defaultOAuthUAVersion = extractCLIVersion(DefaultHeaders["User-Agent"])

// DefaultOAuthUAVersion is the CLI version implied by the OAuth fingerprint UA
// (DefaultHeaders["User-Agent"]). It selects the metadata.user_id wire format.
func DefaultOAuthUAVersion() string {
	return defaultOAuthUAVersion
}

func extractCLIVersion(ua string) string {
	m := claudeCLIVersionRe.FindStringSubmatch(ua)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// OAuthClientID returns a STABLE 64-hex device/client id for a subscription. Anthropic
// correlates a stable device id per account; deriving it deterministically from the
// account_uuid (or channel id when absent) keeps it stable across restarts without a
// fingerprint store — unlike sub2api's random-then-cached id, this needs no persistence.
func OAuthClientID(accountUUID string, channelID int) string {
	seed := strings.TrimSpace(accountUUID)
	if seed == "" {
		seed = fmt.Sprintf("channel:%d", channelID)
	}
	sum := sha256.Sum256([]byte("claude-oauth-device:" + seed))
	return hex.EncodeToString(sum[:]) // 64 hex chars
}

// BuildOAuthMetadataUserID builds the metadata.user_id to inject when the client did
// NOT supply one. Returns "" when the body already carries a non-empty user_id (the
// client's value is then left untouched). deviceID is stable per subscription; sessionID
// is stable per conversation (seeded from the first message) for upstream cache affinity.
func BuildOAuthMetadataUserID(body []byte, accountUUID string, channelID int, uaVersion string) string {
	existing := gjson.GetBytes(body, "metadata.user_id")
	if existing.Exists() && existing.Type == gjson.String && strings.TrimSpace(existing.String()) != "" {
		return ""
	}
	deviceID := OAuthClientID(accountUUID, channelID)
	sessionSeed := strings.TrimSpace(accountUUID) + "::" + firstMessageFingerprint(body)
	sessionID := uuidV4FromSeed(sessionSeed)
	return formatMetadataUserID(deviceID, strings.TrimSpace(accountUUID), sessionID, uaVersion)
}

// firstMessageFingerprint returns a stable hash of the first message (constant across the
// turns of one conversation, since history only grows after it). Falls back to the model.
func firstMessageFingerprint(body []byte) string {
	if first := gjson.GetBytes(body, "messages.0"); first.Exists() {
		sum := sha256.Sum256([]byte(first.Raw))
		return hex.EncodeToString(sum[:8])
	}
	if m := gjson.GetBytes(body, "model"); m.Exists() {
		return m.String()
	}
	return "default"
}

type jsonUserID struct {
	DeviceID    string `json:"device_id"`
	AccountUUID string `json:"account_uuid"`
	SessionID   string `json:"session_id"`
}

func formatMetadataUserID(deviceID, accountUUID, sessionID, uaVersion string) string {
	if isNewMetadataFormatVersion(uaVersion) {
		b, err := json.Marshal(jsonUserID{DeviceID: deviceID, AccountUUID: accountUUID, SessionID: sessionID})
		if err == nil {
			return string(b)
		}
	}
	return "user_" + deviceID + "_account_" + accountUUID + "_session_" + sessionID
}

// uuidV4FromSeed derives a deterministic UUIDv4-formatted string from a seed.
func uuidV4FromSeed(seed string) string {
	h := sha256.Sum256([]byte(seed))
	b := h[:16]
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func isNewMetadataFormatVersion(version string) bool {
	if version == "" {
		return false
	}
	return compareVersions(version, newMetadataFormatMinVersion) >= 0
}

func compareVersions(a, b string) int {
	ap, bp := parseSemver(a), parseSemver(b)
	for i := 0; i < 3; i++ {
		if ap[i] < bp[i] {
			return -1
		}
		if ap[i] > bp[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	res := [3]int{}
	for i := 0; i < len(parts) && i < 3; i++ {
		if n, err := strconv.Atoi(parts[i]); err == nil {
			res[i] = n
		}
	}
	return res
}
