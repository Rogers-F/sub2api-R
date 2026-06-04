package handler

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// relayMaxRequestBodyBytes caps the chat request body forwarded upstream.
const relayMaxRequestBodyBytes = 4 << 20 // 4MB

// relayIdleTimeout cancels the upstream stream if no bytes are received within
// this window, so a hung upstream (with the client still connected) cannot hold
// a goroutine/connection indefinitely. Reset on every chunk of activity.
const relayIdleTimeout = 120 * time.Second

// relayResponseHeaderTimeout bounds the time to receive the upstream response
// headers (the idle watchdog only covers the body phase, which starts after
// headers arrive). Together they prevent a hang before OR during streaming.
const relayResponseHeaderTimeout = 60 * time.Second

// NewAPIRelayHandler is a same-origin BFF that relays chat streaming from the
// logged-in (JWT) user to an external new-api instance using the per-user relay
// token that xsjh auto-provisions.
//
// Hard invariants:
//   - Only ever uses the JWT user's OWN provisioned token. No token/user/model
//     routing override is accepted from the client body (standard chat fields
//     like model/messages pass through untouched).
//   - Requires the feature flag chat.provider_mode == "newapi_bff"; otherwise
//     the endpoints behave as if absent (404).
//   - Requires >=1 active subscription (server-side authorization gate).
//   - The relay key is never written to logs or response headers.
type NewAPIRelayHandler struct {
	cfg                 *config.Config
	provision           *service.NewAPIProvisionService
	subscriptionService *service.SubscriptionService
	httpClient          *http.Client
}

// NewNewAPIRelayHandler constructs the relay handler.
func NewNewAPIRelayHandler(
	cfg *config.Config,
	provision *service.NewAPIProvisionService,
	subscriptionService *service.SubscriptionService,
) *NewAPIRelayHandler {
	// No overall client timeout: chat streaming is long-lived. Cancellation is
	// driven by the request context (client disconnect) and the idle watchdog.
	// A response-header timeout bounds the pre-streaming (connect/headers) phase.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = relayResponseHeaderTimeout
	return &NewAPIRelayHandler{
		cfg:                 cfg,
		provision:           provision,
		subscriptionService: subscriptionService,
		httpClient:          &http.Client{Transport: tr},
	}
}

// enabled reports whether the BFF feature is active.
func (h *NewAPIRelayHandler) enabled() bool {
	return h.cfg != nil && h.cfg.Chat.ProviderMode == config.ChatProviderModeNewAPIBFF && h.cfg.NewAPI.BaseURL != ""
}

// ChatCompletions handles POST /api/v1/newapi/chat/completions.
func (h *NewAPIRelayHandler) ChatCompletions(c *gin.Context) {
	h.relay(c, http.MethodPost, "/v1/chat/completions")
}

// Models handles GET /api/v1/newapi/models.
func (h *NewAPIRelayHandler) Models(c *gin.Context) {
	h.relay(c, http.MethodGet, "/v1/models")
}

func (h *NewAPIRelayHandler) relay(c *gin.Context, method, upstreamPath string) {
	// Feature gate: behave as if the endpoint does not exist when off.
	if !h.enabled() {
		response.NotFound(c, "Not found")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	// Authorization gate: require at least one active subscription.
	// The frontend groups.length is NOT a security check.
	subs, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if len(subs) == 0 {
		response.Forbidden(c, "No active subscription")
		return
	}

	// Resolve the user's own relay key (provisioning on first use).
	key, err := h.provision.GetTokenKey(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Derive a cancelable context from the request so BOTH a client disconnect
	// (request context done) AND an idle-upstream watchdog can cancel the stream.
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Build the upstream request bound to that context.
	var body io.Reader
	if method == http.MethodPost && c.Request.Body != nil {
		body = io.LimitReader(c.Request.Body, relayMaxRequestBodyBytes)
	}
	upstreamURL := h.cfg.NewAPI.BaseURL + upstreamPath
	req, err := http.NewRequestWithContext(ctx, method, upstreamURL, body)
	if err != nil {
		response.InternalError(c, "Failed to build upstream request")
		return
	}
	// Only the user's own token is ever used. The key is never logged.
	req.Header.Set("Authorization", "Bearer "+key)
	if accept := c.GetHeader("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}
	if method == http.MethodPost {
		ct := c.ContentType()
		if ct == "" {
			ct = "application/json"
		}
		req.Header.Set("Content-Type", ct)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		// Client cancellation is expected and not a server error.
		if c.Request.Context().Err() != nil {
			return
		}
		response.Error(c, http.StatusBadGateway, "Upstream request failed")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Propagate content type so SSE/JSON are handled correctly downstream.
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Status(resp.StatusCode)

	// Idle watchdog: cancel the upstream if no bytes arrive within the window.
	idle := time.AfterFunc(relayIdleTimeout, cancel)
	defer idle.Stop()

	flusher, _ := c.Writer.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			idle.Reset(relayIdleTimeout) // activity -> push back the idle deadline
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr != nil {
			// io.EOF (or context cancellation, incl. idle timeout) ends the stream.
			return
		}
	}
}
