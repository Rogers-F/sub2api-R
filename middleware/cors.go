package middleware

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// relayCORSHeaders are the request headers browser relay / API-key clients send
// (Anthropic / OpenAI / Gemini conventions), plus the common web headers.
var relayCORSHeaders = []string{
	"Authorization", "Content-Type", "Accept", "Accept-Language", "Cache-Control",
	"X-Requested-With", "New-Api-User",
	"x-api-key", "x-goog-api-key", "anthropic-version", "anthropic-beta",
	"openai-organization", "openai-project",
}

// CORS is the relay / API-key surface CORS (relay global, /v1, /pg, dashboard,
// /api/usage, /api/log/token). It NEVER pairs AllowAllOrigins with credentials
// (that illegal combo lets any site send cookie-bearing cross-origin requests).
// These clients authenticate via headers (API key), not the session cookie, so
// credentials are off here.
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = false
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = relayCORSHeaders
	return cors.New(config)
}

// SessionCORS is the cookie-session surface CORS, mounted on the /api group.
// By default (no CORS_ALLOW_ORIGINS) it is a no-op: production serves the SPA
// same-origin so no CORS is needed and the credentialed-wildcard hole stays
// closed. When CORS_ALLOW_ORIGINS is set (cross-origin dev), it enables
// credentialed CORS for exactly those origins.
func SessionCORS() gin.HandlerFunc {
	origins := parseAllowedOrigins(common.GetEnvOrDefaultString("CORS_ALLOW_ORIGINS", ""))
	if len(origins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	config := cors.DefaultConfig()
	config.AllowOrigins = origins
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"New-Api-User", "Content-Type", "Authorization", "Accept", "Accept-Language"}
	return cors.New(config)
}

// parseAllowedOrigins splits a comma list into exact origins, dropping blanks,
// "*", and normalizing a trailing slash. A credentialed allowlist must never
// contain a wildcard.
func parseAllowedOrigins(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		o := strings.TrimRight(strings.TrimSpace(p), "/")
		if o == "" || o == "*" {
			continue
		}
		out = append(out, o)
	}
	return out
}

func PoweredBy() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-New-Api-Version", common.Version)
		c.Next()
	}
}
