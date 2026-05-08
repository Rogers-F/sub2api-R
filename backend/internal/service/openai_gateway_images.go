package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func (s *OpenAIGatewayService) ForwardImageRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	contentType string,
	endpointPath string,
	originalModel string,
	imageSize string,
) (*OpenAIForwardResult, error) {
	if account == nil {
		return nil, fmt.Errorf("openai image forward: account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("openai image forward: account type %s is unsupported", account.Type)
	}

	startTime := time.Now()
	mappedModel, _ := account.ResolveMappedModel(originalModel)
	if strings.TrimSpace(mappedModel) == "" {
		mappedModel = originalModel
	}

	requestBody := body
	requestContentType := strings.TrimSpace(contentType)
	if mappedModel != originalModel {
		rewrittenBody, rewrittenContentType, err := rewriteOpenAIImageRequestModel(body, requestContentType, mappedModel)
		if err != nil {
			return nil, err
		}
		requestBody = rewrittenBody
		requestContentType = rewrittenContentType
	}

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	setOpsUpstreamRequestBody(c, requestBody)

	upstreamReq, err := s.buildUpstreamImageRequest(ctx, c, account, requestBody, requestContentType, endpointPath, token)
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"type":    "upstream_error",
				"message": "Upstream request failed",
			},
		})
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
			})

			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			s.handleFailoverSideEffects(ctx, resp, account)
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode),
			}
		}

		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		return s.handleErrorResponse(ctx, resp, c, account, requestBody)
	}
	defer func() { _ = resp.Body.Close() }()

	usage, imageCount, err := s.handleOpenAIImageNonStreamingResponse(resp, c)
	if err != nil {
		return nil, err
	}

	return &OpenAIForwardResult{
		RequestID:     resp.Header.Get("x-request-id"),
		Usage:         usage,
		Model:         originalModel,
		UpstreamModel: mappedModel,
		ImageCount:    imageCount,
		ImageSize:     strings.TrimSpace(imageSize),
		Duration:      time.Since(startTime),
	}, nil
}

func (s *OpenAIGatewayService) buildUpstreamImageRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	contentType string,
	endpointPath string,
	token string,
) (*http.Request, error) {
	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	targetURL := buildOpenAIImagesURL(validatedURL, endpointPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("authorization", "Bearer "+token)
	req.Header.Set("accept", "application/json")
	if trimmedContentType := strings.TrimSpace(contentType); trimmedContentType != "" {
		req.Header.Set("content-type", trimmedContentType)
	} else {
		req.Header.Set("content-type", applicationJSONContentType)
	}

	if c != nil && c.Request != nil {
		if beta := strings.TrimSpace(c.GetHeader("OpenAI-Beta")); beta != "" {
			req.Header.Set("OpenAI-Beta", beta)
		}
		if acceptLanguage := strings.TrimSpace(c.GetHeader("Accept-Language")); acceptLanguage != "" {
			req.Header.Set("Accept-Language", acceptLanguage)
		}
		if userAgent := strings.TrimSpace(c.GetHeader("User-Agent")); userAgent != "" {
			req.Header.Set("User-Agent", userAgent)
		}
	}

	if customUA := strings.TrimSpace(account.GetOpenAIUserAgent()); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}

	return req, nil
}

func (s *OpenAIGatewayService) handleOpenAIImageNonStreamingResponse(resp *http.Response, c *gin.Context) (OpenAIUsage, int, error) {
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
			c.JSON(http.StatusBadGateway, gin.H{
				"error": gin.H{
					"type":    "upstream_error",
					"message": "Upstream response too large",
				},
			})
		}
		return OpenAIUsage{}, 0, err
	}

	usage, imageCount := extractOpenAIImageResponseMeta(body)

	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := resolveNonStreamJSONContentType(c, resp.Header.Get("Content-Type"), applicationJSONContentType)
	c.Data(resp.StatusCode, contentType, body)

	return usage, imageCount, nil
}

func buildOpenAIImagesURL(baseURL, endpointPath string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedEndpoint := "/" + strings.TrimLeft(strings.TrimSpace(endpointPath), "/")
	if trimmedBase == "" || trimmedEndpoint == "/" {
		return trimmedBase
	}
	if strings.HasSuffix(trimmedBase, trimmedEndpoint) {
		return trimmedBase
	}
	if strings.HasSuffix(trimmedBase, "/v1") && strings.HasPrefix(trimmedEndpoint, "/v1/") {
		return trimmedBase + strings.TrimPrefix(trimmedEndpoint, "/v1")
	}
	return trimmedBase + trimmedEndpoint
}

func rewriteOpenAIImageRequestModel(body []byte, contentType, mappedModel string) ([]byte, string, error) {
	if strings.TrimSpace(mappedModel) == "" {
		return body, contentType, nil
	}

	trimmedContentType := strings.TrimSpace(contentType)
	if trimmedContentType == "" || strings.Contains(strings.ToLower(trimmedContentType), "application/json") {
		if len(body) == 0 || !gjson.ValidBytes(body) {
			return body, contentType, nil
		}
		nextBody, err := sjson.SetBytes(body, "model", mappedModel)
		if err != nil {
			return nil, "", fmt.Errorf("rewrite openai image request model: %w", err)
		}
		if trimmedContentType == "" {
			trimmedContentType = applicationJSONContentType
		}
		return nextBody, trimmedContentType, nil
	}

	mediaType, params, err := mime.ParseMediaType(trimmedContentType)
	if err != nil {
		return nil, "", fmt.Errorf("parse openai image request content-type: %w", err)
	}
	if !strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		return body, contentType, nil
	}

	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return nil, "", fmt.Errorf("parse openai image request content-type: missing multipart boundary")
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	modelRewritten := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("rewrite openai image multipart body: %w", err)
		}

		header := cloneMultipartHeader(part.Header)
		dst, err := writer.CreatePart(header)
		if err != nil {
			return nil, "", fmt.Errorf("rewrite openai image multipart body: %w", err)
		}

		if part.FormName() == "model" && part.FileName() == "" {
			if _, err := io.WriteString(dst, mappedModel); err != nil {
				return nil, "", fmt.Errorf("rewrite openai image multipart body: %w", err)
			}
			modelRewritten = true
			continue
		}

		if _, err := io.Copy(dst, part); err != nil {
			return nil, "", fmt.Errorf("rewrite openai image multipart body: %w", err)
		}
	}

	if !modelRewritten {
		if err := writer.WriteField("model", mappedModel); err != nil {
			return nil, "", fmt.Errorf("rewrite openai image multipart body: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("rewrite openai image multipart body: %w", err)
	}

	return buf.Bytes(), writer.FormDataContentType(), nil
}

func cloneMultipartHeader(src textproto.MIMEHeader) textproto.MIMEHeader {
	if src == nil {
		return textproto.MIMEHeader{}
	}

	dst := make(textproto.MIMEHeader, len(src))
	for key, values := range src {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
	return dst
}

func extractOpenAIImageResponseMeta(body []byte) (OpenAIUsage, int) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return OpenAIUsage{}, 0
	}

	imageCount := len(gjson.GetBytes(body, "data").Array())
	return extractOpenAIUsageAtPath(body, "usage"), imageCount
}

func firstPositiveGJSONInt(values ...gjson.Result) int {
	for _, value := range values {
		if value.Int() > 0 {
			return int(value.Int())
		}
	}
	return 0
}
