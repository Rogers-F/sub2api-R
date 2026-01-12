//go:build embed

package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"html"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

//go:embed all:dist
var frontendFS embed.FS

// PublicSettingsProvider is an interface to fetch public settings
type PublicSettingsProvider interface {
	GetPublicSettingsForInjection(ctx context.Context) (any, error)
}

// FrontendServer serves the embedded frontend with settings injection
type FrontendServer struct {
	distFS     fs.FS
	fileServer http.Handler
	baseHTML   []byte
	cache      *HTMLCache
	settings   PublicSettingsProvider
}

// NewFrontendServer creates a new frontend server with settings injection
func NewFrontendServer(settingsProvider PublicSettingsProvider) (*FrontendServer, error) {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return nil, err
	}

	// Read base HTML once
	file, err := distFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	baseHTML, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	cache := NewHTMLCache()
	cache.SetBaseHTML(baseHTML)

	return &FrontendServer{
		distFS:     distFS,
		fileServer: http.FileServer(http.FS(distFS)),
		baseHTML:   baseHTML,
		cache:      cache,
		settings:   settingsProvider,
	}, nil
}

// InvalidateCache invalidates the HTML cache (call when settings change)
func (s *FrontendServer) InvalidateCache() {
	if s != nil && s.cache != nil {
		s.cache.Invalidate()
	}
}

// Middleware returns the Gin middleware handler
func (s *FrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API routes
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/v1/") ||
			strings.HasPrefix(path, "/v1beta/") ||
			strings.HasPrefix(path, "/antigravity/") ||
			strings.HasPrefix(path, "/setup/") ||
			path == "/health" ||
			path == "/responses" {
			c.Next()
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// For index.html or SPA routes, serve with injected settings
		if cleanPath == "index.html" || !s.fileExists(cleanPath) {
			s.serveIndexHTML(c)
			return
		}

		// Serve static files normally
		s.fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func (s *FrontendServer) fileExists(path string) bool {
	file, err := s.distFS.Open(path)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func (s *FrontendServer) serveIndexHTML(c *gin.Context) {
	// Check cache first
	cached := s.cache.Get()
	if cached != nil {
		// Check If-None-Match for 304 response
		if match := c.GetHeader("If-None-Match"); match == cached.ETag {
			c.Status(http.StatusNotModified)
			c.Abort()
			return
		}

		c.Header("ETag", cached.ETag)
		c.Header("Cache-Control", "no-cache") // Must revalidate
		c.Data(http.StatusOK, "text/html; charset=utf-8", cached.Content)
		c.Abort()
		return
	}

	// Cache miss - fetch settings and render
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	settings, err := s.settings.GetPublicSettingsForInjection(ctx)
	if err != nil {
		// Fallback: serve without injection
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		// Fallback: serve without injection
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	rendered := s.injectSettings(settingsJSON)
	s.cache.Set(rendered, settingsJSON)

	cached = s.cache.Get()
	if cached != nil {
		c.Header("ETag", cached.ETag)
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", rendered)
	c.Abort()
}

func (s *FrontendServer) injectSettings(settingsJSON []byte) []byte {
	rendered := s.baseHTML

	// Server-side title injection to prevent flash of default title
	siteName := strings.TrimSpace(gjson.GetBytes(settingsJSON, "site_name").String())
	if siteName != "" && siteName != "Sub2API" {
		// Limit length to prevent abuse (256 chars is generous for a site name)
		if len(siteName) > 256 {
			siteName = siteName[:256]
		}
		escapedTitle := html.EscapeString(siteName) + " - AI API Gateway"
		rendered = replaceHTMLTitle(rendered, escapedTitle)
	}

	// Build injection content
	var injection bytes.Buffer

	// Inject preload link for custom logo to eliminate flash (browser starts downloading immediately)
	siteLogo := strings.TrimSpace(gjson.GetBytes(settingsJSON, "site_logo").String())
	if siteLogo != "" {
		// Protocol validation (defense in depth) - only allow safe URL schemes
		// Note: Reject protocol-relative URLs (//) to align with frontend sanitizeUrl
		isValidProtocol := strings.HasPrefix(strings.ToLower(siteLogo), "http://") ||
			strings.HasPrefix(strings.ToLower(siteLogo), "https://") ||
			(strings.HasPrefix(siteLogo, "/") && !strings.HasPrefix(siteLogo, "//")) ||
			strings.HasPrefix(strings.ToLower(siteLogo), "data:image/")

		if isValidProtocol {
			// Limit length to prevent abuse (2048 for URLs, 410KB for data URIs)
			// Note: Frontend allows 300KB file uploads, but base64 encoding inflates by ~4/3
			// 300KB * 4/3 ≈ 400KB, plus data URI header overhead → 410KB
			maxLen := 2048
			siteLogoLower := strings.ToLower(siteLogo)
			if strings.HasPrefix(siteLogoLower, "data:") {
				maxLen = 410 * 1024 // 410KB for base64 data URIs (accounts for base64 inflation)
			}
			if len(siteLogo) <= maxLen {
				escapedLogo := html.EscapeString(siteLogo)
				// Only preload URL-based logos (data URIs are already inline, no network request needed)
				if !strings.HasPrefix(siteLogoLower, "data:") {
					injection.WriteString(`<link rel="preload" href="` + escapedLogo + `" as="image">`)
				}
				// Set as favicon to eliminate favicon flash
				injection.WriteString(`<link rel="icon" href="` + escapedLogo + `">`)
			}
		}
	}

	// Add the config script
	injection.WriteString(`<script>window.__APP_CONFIG__=` + string(settingsJSON) + `;</script>`)

	// Inject before </head>
	headClose := []byte("</head>")
	return bytes.Replace(rendered, headClose, append(injection.Bytes(), headClose...), 1)
}

// replaceHTMLTitle replaces the content of the <title> tag.
// It searches only the first 2KB for performance (title is always in <head>).
// Handles case-insensitive matching without full ToLower allocation.
func replaceHTMLTitle(page []byte, newTitle string) []byte {
	// Limit search to first 2KB (title is always in <head>)
	searchLimit := 2048
	if len(page) < searchLimit {
		searchLimit = len(page)
	}
	searchArea := page[:searchLimit]

	// Find <title (case-insensitive)
	openIdx := indexCaseInsensitive(searchArea, []byte("<title"))
	if openIdx == -1 {
		return page
	}

	// Find > after <title
	gtIdx := bytes.IndexByte(searchArea[openIdx:], '>')
	if gtIdx == -1 {
		return page
	}
	gtIdx += openIdx // absolute position

	// Find </title> (case-insensitive)
	closeIdx := indexCaseInsensitive(searchArea[gtIdx:], []byte("</title>"))
	if closeIdx == -1 {
		return page
	}
	closeIdx += gtIdx // absolute position

	// Build result: before title content + new title + after title content
	titleStart := gtIdx + 1
	titleEnd := closeIdx

	result := make([]byte, 0, len(page)-titleEnd+titleStart+len(newTitle))
	result = append(result, page[:titleStart]...)
	result = append(result, newTitle...)
	result = append(result, page[titleEnd:]...)
	return result
}

// indexCaseInsensitive finds needle in haystack (case-insensitive).
// Only used for short patterns like "<title" and "</title>".
func indexCaseInsensitive(haystack, needle []byte) int {
	needleLower := bytes.ToLower(needle)
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if bytes.EqualFold(haystack[i:i+len(needle)], needleLower) {
			return i
		}
	}
	return -1
}

// ServeEmbeddedFrontend returns a middleware for serving embedded frontend
// This is the legacy function for backward compatibility when no settings provider is available
func ServeEmbeddedFrontend() gin.HandlerFunc {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic("failed to get dist subdirectory: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(distFS))

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/v1/") ||
			strings.HasPrefix(path, "/v1beta/") ||
			strings.HasPrefix(path, "/antigravity/") ||
			strings.HasPrefix(path, "/setup/") ||
			path == "/health" ||
			path == "/responses" {
			c.Next()
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if file, err := distFS.Open(cleanPath); err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		serveIndexHTML(c, distFS)
	}
}

func serveIndexHTML(c *gin.Context, fsys fs.FS) {
	file, err := fsys.Open("index.html")
	if err != nil {
		c.String(http.StatusNotFound, "Frontend not found")
		c.Abort()
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read index.html")
		c.Abort()
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func HasEmbeddedFrontend() bool {
	_, err := frontendFS.ReadFile("dist/index.html")
	return err == nil
}
