package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) ImagesGenerations(c *gin.Context) {
	h.handleImages(c, EndpointImagesGenerations)
}

func (h *OpenAIGatewayHandler) ImagesEdits(c *gin.Context) {
	h.handleImages(c, EndpointImagesEdits)
}

func (h *OpenAIGatewayHandler) handleImages(c *gin.Context, endpoint string) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	reqLog := requestLogger(
		c,
		"handler.openai_gateway.images",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.String("endpoint", endpoint),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	reqModel, imageSize, err := extractOpenAIImageRequestMeta(contentType, body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	if reqModel == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqLog = reqLog.With(zap.String("model", reqModel), zap.String("image_size", imageSize))

	setOpsRequestContext(c, reqModel, false, body)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("openai_images.billing_eligibility_check_failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		reqLog.Debug("openai_images.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithScheduler(
			c.Request.Context(),
			apiKey.GroupID,
			"",
			"",
			reqModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
		)
		if err != nil {
			reqLog.Warn("openai_images.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable", streamStarted)
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
			} else {
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available OpenAI API key accounts for image endpoints", streamStarted)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
			return
		}

		account := selection.Account
		if account.Type != service.AccountTypeAPIKey {
			failedAccountIDs[account.ID] = struct{}{}
			reqLog.Info("openai_images.skip_unsupported_account_type",
				zap.Int64("account_id", account.ID),
				zap.String("account_type", account.Type),
			)
			continue
		}

		reqLog.Debug("openai_images.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		_ = scheduleDecision
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, "", selection, false, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		result, err := h.gatewayService.ForwardImageRequest(c.Request.Context(), c, account, body, contentType, endpoint, reqModel, imageSize)

		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)

		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				if failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai_images.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-c.Request.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}

				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhausted(c, failoverErr, streamStarted)
					return
				}
				switchCount++
				reqLog.Warn("openai_images.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}

			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
			reqLog.Warn("openai_images.forward_failed",
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			)
			return
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)

		userAgent := c.GetHeader("User-Agent")
		clientIP := ip.GetClientIP(c)
		requestPayloadHash := service.HashUsageRequestPayload(body)

		h.submitUsageRecordTask(func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             apiKey,
				User:               apiKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    GetInboundEndpoint(c),
				UpstreamEndpoint:   GetUpstreamEndpoint(c, account.Platform),
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.images"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", apiKey.ID),
					zap.Any("group_id", apiKey.GroupID),
					zap.String("model", reqModel),
					zap.Int64("account_id", account.ID),
				).Error("openai_images.record_usage_failed", zap.Error(err))
			}
		})

		reqLog.Debug("openai_images.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func extractOpenAIImageRequestMeta(contentType string, body []byte) (string, string, error) {
	trimmedContentType := strings.TrimSpace(contentType)
	if trimmedContentType == "" || strings.Contains(strings.ToLower(trimmedContentType), "application/json") {
		model := strings.TrimSpace(gjsonGetString(body, "model"))
		size := strings.TrimSpace(gjsonGetString(body, "size"))
		if size == "" {
			size = strings.TrimSpace(gjsonGetString(body, "image_size"))
		}
		return model, size, nil
	}

	mediaType, params, err := mime.ParseMediaType(trimmedContentType)
	if err != nil {
		return "", "", errors.New("failed to parse request body")
	}
	if !strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		return "", "", errors.New("unsupported image request content type")
	}

	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return "", "", errors.New("invalid multipart form data")
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var model string
	var size string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", errors.New("invalid multipart form data")
		}
		if part.FileName() != "" {
			continue
		}

		name := strings.TrimSpace(part.FormName())
		if name != "model" && name != "size" && name != "image_size" {
			continue
		}

		valueBytes, err := io.ReadAll(io.LimitReader(part, 64<<10))
		if err != nil {
			return "", "", errors.New("invalid multipart form data")
		}
		value := strings.TrimSpace(string(valueBytes))
		switch name {
		case "model":
			model = value
		case "size", "image_size":
			size = value
		}
	}

	return model, size, nil
}

func gjsonGetString(body []byte, path string) string {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return ""
	}
	return gjson.GetBytes(body, path).String()
}
