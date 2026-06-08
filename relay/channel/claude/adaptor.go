package claude

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	requestURL := fmt.Sprintf("%s/v1/messages", info.ChannelBaseUrl)
	if !shouldAppendClaudeBetaQuery(info) {
		return requestURL, nil
	}

	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return "", err
	}
	query := parsedURL.Query()
	query.Set("beta", "true")
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func shouldAppendClaudeBetaQuery(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	if info.IsClaudeBetaQuery {
		return true
	}
	if info.ChannelOtherSettings.ClaudeBetaQuery {
		return true
	}
	return false
}

func CommonClaudeHeadersOperation(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) {
	// common headers operation
	anthropicBeta := c.Request.Header.Get("anthropic-beta")
	if anthropicBeta != "" {
		req.Set("anthropic-beta", anthropicBeta)
	}
	model_setting.GetClaudeSettings().WriteHeaders(info.OriginModelName, req)
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	if info.IsOAuth {
		return setupOAuthRequestHeader(c, req, info)
	}
	req.Set("x-api-key", info.ApiKey)
	anthropicVersion := c.Request.Header.Get("anthropic-version")
	if anthropicVersion == "" {
		anthropicVersion = "2023-06-01"
	}
	req.Set("anthropic-version", anthropicVersion)
	CommonClaudeHeadersOperation(c, req, info)
	return nil
}

// setupOAuthRequestHeader builds the header set for a Claude(Anthropic) OAuth
// subscription request: a freshly-ensured Bearer token (NOT x-api-key), the exact
// Claude CLI fingerprint, and the mimic anthropic-beta set (claude-code beta dropped).
// The final outbound sanitizer (api_request.go) re-asserts the Bearer and enforces the
// allowlist after any header overrides run, so injection here cannot be undone.
func setupOAuthRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	accessToken, err := service.EnsureClaudeOAuthFresh(c.Request.Context(), info.ChannelId, info.ApiKey, info.ChannelSetting.Proxy)
	if err != nil {
		return fmt.Errorf("claude oauth: %w", err)
	}

	bearer := "Bearer " + accessToken
	req.Del("x-api-key")
	req.Set("Authorization", bearer)
	info.OAuthOutboundBearer = bearer
	info.OAuthAllowedHeaders = OAuthAllowedHeaders

	anthropicVersion := c.Request.Header.Get("anthropic-version")
	if anthropicVersion == "" {
		anthropicVersion = "2023-06-01"
	}

	// anthropic-beta: the /v1/messages mimic branch drops the claude-code beta; haiku
	// omits it as well. Both resolve to [oauth, interleaved-thinking].
	model := strings.ToLower(info.UpstreamModelName)
	if model == "" {
		model = strings.ToLower(info.OriginModelName)
	}
	betaHeader := strings.Join(MimicMessageBetas, ",")
	if strings.Contains(model, "haiku") {
		betaHeader = HaikuBetaHeader
	}

	// Force the canonical Claude CLI / Stainless fingerprint + version/beta. These clients
	// are not real CLIs, so the OAuth path mimics the fingerprint EXACTLY (force-set, not
	// fill-if-empty). The same set is recorded as OAuthForcedHeaders so the final outbound
	// sanitizer re-asserts it AFTER channel header overrides run (those header names are
	// allowlisted and an override must not be able to break the mimic).
	forced := make(map[string]string, len(DefaultHeaders)+2)
	for k, v := range DefaultHeaders {
		forced[k] = v
	}
	forced["anthropic-version"] = anthropicVersion
	forced["anthropic-beta"] = betaHeader
	for k, v := range forced {
		req.Set(k, v)
	}
	info.OAuthForcedHeaders = forced
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return RequestOpenAI2ClaudeMessage(c, *request)
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	info.FinalRequestRelayFormat = types.RelayFormatClaude
	if info.IsStream {
		return ClaudeStreamHandler(c, resp, info)
	} else {
		return ClaudeHandler(c, resp, info)
	}
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
