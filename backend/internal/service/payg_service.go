package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

const shouqianbaAPIBase = "https://vsi-api.shouqianba.com"

type shouqianbaRequestMeta struct {
	HTTPStatus int
	Body       string
}

type paygPythonJSONField struct {
	key string
	raw string
}

type PaygService struct {
	repo                 PaygOrderRepository
	userRepo             UserRepository
	settingService       *SettingService
	referralService      *ReferralService
	billingCache         BillingCache
	authCacheInvalidator APIKeyAuthCacheInvalidator
	entClient            *dbent.Client
	httpClient           *http.Client
}

func NewPaygService(
	repo PaygOrderRepository,
	userRepo UserRepository,
	settingService *SettingService,
	referralService *ReferralService,
	billingCache BillingCache,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	entClient *dbent.Client,
) *PaygService {
	return &PaygService{
		repo:                 repo,
		userRepo:             userRepo,
		settingService:       settingService,
		referralService:      referralService,
		billingCache:         billingCache,
		authCacheInvalidator: authCacheInvalidator,
		entClient:            entClient,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type PaygPrecreateResult struct {
	Order  *PaygOrder `json:"order"`
	QRCode string     `json:"qr_code"`
}

func (s *PaygService) GetWallet(ctx context.Context, userID int64) (*PaygWallet, error) {
	cfg := s.settingService.GetPaygSettings(ctx)

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	summary, err := s.repo.GetUserSummary(ctx, userID, 50)
	if err != nil {
		return nil, fmt.Errorf("get payg wallet summary: %w", err)
	}

	totalConsumption := 0.0
	if summary.TotalCreditedAmount > user.Balance {
		totalConsumption = roundCurrency(summary.TotalCreditedAmount - user.Balance)
	}

	return &PaygWallet{
		Enabled:             cfg.Enabled,
		Balance:             roundCurrency(user.Balance),
		ExchangeRate:        cfg.ExchangeRate,
		FixedAmountOptions:  cfg.FixedAmountOptions,
		TotalPaidAmount:     roundCurrency(summary.TotalPaidAmount),
		TotalCreditedAmount: roundCurrency(summary.TotalCreditedAmount),
		TotalConsumption:    totalConsumption,
		Orders:              summary.Orders,
	}, nil
}

func (s *PaygService) GetAdminWallet(ctx context.Context) (*PaygAdminWallet, error) {
	cfg := s.settingService.GetPaygSettings(ctx)

	summary, err := s.repo.GetAdminSummary(ctx, 100, 100)
	if err != nil {
		return nil, fmt.Errorf("get admin payg summary: %w", err)
	}

	return &PaygAdminWallet{
		Enabled:             cfg.Enabled,
		TotalOrders:         summary.TotalOrders,
		PaidOrders:          summary.PaidOrders,
		PendingOrders:       summary.PendingOrders,
		TotalPaidAmount:     roundCurrency(summary.TotalPaidAmount),
		TotalCreditedAmount: roundCurrency(summary.TotalCreditedAmount),
		Users:               summary.Users,
		Orders:              summary.Orders,
	}, nil
}

func (s *PaygService) HasPendingOrders(ctx context.Context) (bool, error) {
	return s.repo.HasPendingOrders(ctx)
}

func (s *PaygService) ValidateCallbackToken(token string) error {
	expected := strings.TrimSpace(s.settingService.GetPaygCallbackToken())
	if expected == "" {
		return ErrPaygCallbackUnconfigured
	}
	token = strings.TrimSpace(token)
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		return ErrPaygCallbackUnauthorized
	}
	return nil
}

func (s *PaygService) Precreate(ctx context.Context, userID int64, amountYuan float64, payway string) (*PaygPrecreateResult, error) {
	cfg := s.settingService.GetPaygSettings(ctx)
	if !cfg.Enabled {
		return nil, ErrPaygDisabled
	}
	if strings.TrimSpace(cfg.TerminalSN) == "" || strings.TrimSpace(cfg.TerminalKey) == "" {
		return nil, ErrPaygProviderNotConfigured
	}

	amountYuan = roundCurrency(amountYuan)
	if amountYuan <= 0 {
		return nil, ErrPaygInvalidAmount
	}

	if payway != PaygPaywayWeChat {
		payway = PaygPaywayAlipay
	}

	clientSN, err := generatePaygClientSN()
	if err != nil {
		return nil, fmt.Errorf("generate payg client_sn: %w", err)
	}

	amountCent := int64(math.Round(amountYuan * 100))
	amountYuanText := formatPaygAmountString(amountYuan)
	subject := buildPaygSubject(s.settingService.GetSiteName(ctx), amountYuanText)
	qrCode, sn, err := s.shouqianbaPrecreate(ctx, cfg, clientSN, amountCent, amountYuanText, payway, subject, userID)
	if err != nil {
		return nil, err
	}

	order := &PaygOrder{
		UserID:       userID,
		ClientSN:     clientSN,
		SN:           sn,
		AmountYuan:   amountYuan,
		AmountCent:   amountCent,
		CreditAmount: roundCurrency(amountYuan * cfg.ExchangeRate),
		Payway:       payway,
		PaywayName:   paywayNameFromCode(payway),
		Status:       PaygOrderStatusPending,
	}
	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create payg order: %w", err)
	}

	return &PaygPrecreateResult{
		Order:  order,
		QRCode: qrCode,
	}, nil
}

func (s *PaygService) QueryOrderForUser(ctx context.Context, userID, orderID int64) (*PaygOrder, error) {
	order, err := s.repo.GetByIDForUser(ctx, orderID, userID)
	if err != nil {
		return nil, err
	}
	if order.Status != PaygOrderStatusPending {
		return order, nil
	}
	syncedOrder, err := s.syncOrderByIdentifiers(ctx, order.SN, order.ClientSN)
	if err != nil {
		if errors.Is(err, ErrPaygAmountMismatch) {
			return nil, err
		}
		log.Printf("[PAYG] query order sync failed, returning local state: order_id=%d err=%v", order.ID, err)
		return order, nil
	}
	return syncedOrder, nil
}

func (s *PaygService) HandleCallback(ctx context.Context, sn, clientSN string) (*PaygOrder, error) {
	sn = strings.TrimSpace(sn)
	clientSN = strings.TrimSpace(clientSN)
	if sn == "" && clientSN == "" {
		return nil, ErrPaygOrderNotFound
	}

	order, err := s.repo.GetByIdentifiers(ctx, sn, clientSN)
	if err == nil {
		if order.Status != PaygOrderStatusPending {
			return order, nil
		}
	} else if !errors.Is(err, ErrPaygOrderNotFound) {
		return nil, err
	}

	return s.syncOrderByIdentifiers(ctx, sn, clientSN)
}

func (s *PaygService) syncOrderByIdentifiers(ctx context.Context, sn, clientSN string) (*PaygOrder, error) {
	if sn == "" && clientSN == "" {
		return nil, ErrPaygOrderNotFound
	}

	cfg := s.settingService.GetPaygSettings(ctx)
	if strings.TrimSpace(cfg.TerminalSN) == "" || strings.TrimSpace(cfg.TerminalKey) == "" {
		return nil, ErrPaygProviderNotConfigured
	}

	providerStatus, err := s.shouqianbaQuery(ctx, cfg, sn, clientSN)
	if err != nil {
		return nil, err
	}

	if providerStatus.SN != "" {
		sn = providerStatus.SN
	}
	if providerStatus.ClientSN != "" {
		clientSN = providerStatus.ClientSN
	}

	order, err := s.repo.GetByIdentifiers(ctx, sn, clientSN)
	if err != nil {
		return nil, err
	}

	order.SN = firstNonEmpty(providerStatus.SN, order.SN)
	order.ClientSN = firstNonEmpty(providerStatus.ClientSN, order.ClientSN)
	order.Payway = firstNonEmpty(providerStatus.Payway, order.Payway)
	order.PaywayName = firstNonEmpty(providerStatus.PaywayName, order.PaywayName)

	if providerStatus.Status != PaygOrderStatusPaid {
		if providerStatus.Status != "" {
			order.Status = providerStatus.Status
		}
		if err := s.repo.UpdateProviderState(ctx, order); err != nil {
			return nil, fmt.Errorf("update payg order state: %w", err)
		}
		return order, nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin payg transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	lockedOrder, err := s.repo.GetForUpdateByIdentifiers(txCtx, sn, clientSN)
	if err != nil {
		return nil, err
	}
	if providerStatus.TotalAmountCent > 0 && lockedOrder.AmountCent > 0 && providerStatus.TotalAmountCent != lockedOrder.AmountCent {
		return nil, ErrPaygAmountMismatch.WithMetadata(map[string]string{
			"local_amount_cent":    strconv.FormatInt(lockedOrder.AmountCent, 10),
			"provider_amount_cent": strconv.FormatInt(providerStatus.TotalAmountCent, 10),
			"order_id":             strconv.FormatInt(lockedOrder.ID, 10),
		})
	}
	if lockedOrder.Status == PaygOrderStatusPaid {
		if err := tx.Rollback(); err != nil {
			log.Printf("[PAYG] rollback paid order transaction failed: order_id=%d err=%v", lockedOrder.ID, err)
		}
		return lockedOrder, nil
	}

	lockedOrder.Status = PaygOrderStatusPaid
	lockedOrder.SN = firstNonEmpty(providerStatus.SN, lockedOrder.SN)
	lockedOrder.ClientSN = firstNonEmpty(providerStatus.ClientSN, lockedOrder.ClientSN)
	lockedOrder.Payway = firstNonEmpty(providerStatus.Payway, lockedOrder.Payway)
	lockedOrder.PaywayName = firstNonEmpty(providerStatus.PaywayName, lockedOrder.PaywayName)
	if providerStatus.PaidAt != nil {
		lockedOrder.PaidAt = providerStatus.PaidAt
	} else {
		now := time.Now()
		lockedOrder.PaidAt = &now
	}

	if err := s.repo.MarkPaid(txCtx, lockedOrder); err != nil {
		return nil, fmt.Errorf("mark payg order paid: %w", err)
	}
	if err := s.userRepo.UpdateBalance(txCtx, lockedOrder.UserID, lockedOrder.CreditAmount); err != nil {
		return nil, fmt.Errorf("credit user balance: %w", err)
	}

	user, userErr := s.userRepo.GetByID(ctx, lockedOrder.UserID)
	if userErr != nil {
		log.Printf("[PAYG] failed to get user for commission: order_id=%d user_id=%d err=%v", lockedOrder.ID, lockedOrder.UserID, userErr)
	} else if s.referralService != nil && user.ReferrerID != nil {
		if _, commissionErr := s.referralService.ProcessCommission(
			txCtx,
			lockedOrder.UserID,
			*user.ReferrerID,
			ReferralSourceTypePaygOrder,
			lockedOrder.ID,
			lockedOrder.CreditAmount,
		); commissionErr != nil {
			log.Printf("[PAYG] failed to process referral commission: order_id=%d err=%v", lockedOrder.ID, commissionErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit payg transaction: %w", err)
	}

	s.invalidateCaches(ctx, lockedOrder.UserID)
	return lockedOrder, nil
}

func (s *PaygService) invalidateCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCache != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.billingCache.InvalidateUserBalance(cacheCtx, userID); err != nil {
				log.Printf("[PAYG] invalidate user balance cache failed: user_id=%d err=%v", userID, err)
			}
		}()
	}
}

func (s *PaygService) shouqianbaPrecreate(
	ctx context.Context,
	cfg *PaygSettings,
	clientSN string,
	amountCent int64,
	amountYuanText string,
	payway string,
	subject string,
	userID int64,
) (qrCode string, sn string, err error) {
	reflectPayload := marshalPythonStyleJSONObject(
		paygPythonRawField("user_id", strconv.FormatInt(userID, 10)),
		paygPythonStringField("amount_yuan", amountYuanText),
	)
	payload := marshalPythonStyleJSONObject(
		paygPythonStringField("terminal_sn", cfg.TerminalSN),
		paygPythonStringField("client_sn", clientSN),
		paygPythonStringField("total_amount", strconv.FormatInt(amountCent, 10)),
		paygPythonStringField("payway", payway),
		paygPythonStringField("subject", subject),
		paygPythonStringField("operator", "system"),
		paygPythonStringField("reflect", reflectPayload),
	)

	var resp struct {
		ResultCode  string `json:"result_code"`
		Error       string `json:"error"`
		BizResponse struct {
			ResultCode string `json:"result_code"`
			Error      string `json:"error"`
			Data       struct {
				QRCode string `json:"qr_code"`
				SN     string `json:"sn"`
			} `json:"data"`
		} `json:"biz_response"`
	}
	meta, err := s.shouqianbaRequest(ctx, cfg, "/upay/v2/precreate", payload, &resp)
	if err != nil {
		return "", "", err
	}
	if resp.ResultCode != "200" || resp.BizResponse.ResultCode != "PRECREATE_SUCCESS" {
		logPaygProviderFailure("precreate", payload, meta, resp.ResultCode, resp.BizResponse.ResultCode, resp.Error, resp.BizResponse.Error)
		return "", "", newPaygProviderRejectedError(meta, resp.ResultCode, resp.BizResponse.ResultCode, resp.Error, resp.BizResponse.Error)
	}
	return strings.TrimSpace(resp.BizResponse.Data.QRCode), strings.TrimSpace(resp.BizResponse.Data.SN), nil
}

func (s *PaygService) shouqianbaQuery(ctx context.Context, cfg *PaygSettings, sn, clientSN string) (*PaygProviderOrderStatus, error) {
	fields := []paygPythonJSONField{
		paygPythonStringField("terminal_sn", cfg.TerminalSN),
	}
	if strings.TrimSpace(sn) != "" {
		fields = append(fields, paygPythonStringField("sn", strings.TrimSpace(sn)))
	} else if strings.TrimSpace(clientSN) != "" {
		fields = append(fields, paygPythonStringField("client_sn", strings.TrimSpace(clientSN)))
	}
	payload := marshalPythonStyleJSONObject(fields...)

	var resp struct {
		ResultCode  string `json:"result_code"`
		Error       string `json:"error"`
		BizResponse struct {
			ResultCode string `json:"result_code"`
			Error      string `json:"error"`
			Data       struct {
				ClientSN    string `json:"client_sn"`
				SN          string `json:"sn"`
				OrderStatus string `json:"order_status"`
				TotalAmount string `json:"total_amount"`
				Payway      string `json:"payway"`
				PaywayName  string `json:"payway_name"`
			} `json:"data"`
		} `json:"biz_response"`
	}
	meta, err := s.shouqianbaRequest(ctx, cfg, "/upay/v2/query", payload, &resp)
	if err != nil {
		return nil, err
	}
	if resp.ResultCode != "200" {
		logPaygProviderFailure("query", payload, meta, resp.ResultCode, resp.BizResponse.ResultCode, resp.Error, resp.BizResponse.Error)
		return nil, newPaygProviderRejectedError(meta, resp.ResultCode, resp.BizResponse.ResultCode, resp.Error, resp.BizResponse.Error)
	}
	if isPaygQueryBizFailure(resp.BizResponse.ResultCode) {
		logPaygProviderFailure("query", payload, meta, resp.ResultCode, resp.BizResponse.ResultCode, resp.Error, resp.BizResponse.Error)
		return nil, newPaygProviderRejectedError(meta, resp.ResultCode, resp.BizResponse.ResultCode, resp.Error, resp.BizResponse.Error)
	}

	totalAmountCent, _ := strconv.ParseInt(strings.TrimSpace(resp.BizResponse.Data.TotalAmount), 10, 64)
	return &PaygProviderOrderStatus{
		ClientSN:        strings.TrimSpace(resp.BizResponse.Data.ClientSN),
		SN:              strings.TrimSpace(resp.BizResponse.Data.SN),
		Status:          normalizePaygOrderStatus(resp.BizResponse.Data.OrderStatus),
		Payway:          strings.TrimSpace(resp.BizResponse.Data.Payway),
		PaywayName:      strings.TrimSpace(resp.BizResponse.Data.PaywayName),
		TotalAmountCent: totalAmountCent,
	}, nil
}

func isPaygQueryBizFailure(resultCode string) bool {
	code := strings.ToUpper(strings.TrimSpace(resultCode))
	if code == "" {
		return false
	}
	return strings.Contains(code, "FAIL") || strings.Contains(code, "ERROR")
}

func (s *PaygService) shouqianbaRequest(ctx context.Context, cfg *PaygSettings, path string, payload string, out any) (*shouqianbaRequestMeta, error) {
	meta := &shouqianbaRequestMeta{}
	sum := md5.Sum([]byte(payload + cfg.TerminalKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shouqianbaAPIBase+path, bytes.NewReader([]byte(payload)))
	if err != nil {
		return nil, fmt.Errorf("create shouqianba request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.TerminalSN+" "+hex.EncodeToString(sum[:]))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send shouqianba request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	meta.HTTPStatus = resp.StatusCode

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return meta, fmt.Errorf("read shouqianba response: %w", err)
	}
	meta.Body = strings.TrimSpace(string(respBody))
	if err := json.Unmarshal(respBody, out); err != nil {
		return meta, fmt.Errorf("decode shouqianba response: status=%d body=%q: %w", meta.HTTPStatus, truncateForLog(meta.Body, 512), err)
	}
	return meta, nil
}

func generatePaygClientSN() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("XS%d%s", time.Now().UnixMilli(), hex.EncodeToString(b)[:6]), nil
}

func paywayNameFromCode(payway string) string {
	switch payway {
	case PaygPaywayWeChat:
		return "微信"
	default:
		return "支付宝"
	}
}

func normalizePaygOrderStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case PaygOrderStatusPaid:
		return PaygOrderStatusPaid
	case PaygOrderStatusClosed:
		return PaygOrderStatusClosed
	default:
		return PaygOrderStatusPending
	}
}

func roundCurrency(v float64) float64 {
	return math.Round(v*100) / 100
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func buildPaygSubject(siteName, amountYuanText string) string {
	name := strings.TrimSpace(siteName)
	if name == "" {
		name = "Sub2API"
	}
	return name + "充值 ¥" + amountYuanText
}

func formatPaygAmountString(amountYuan float64) string {
	return strconv.FormatFloat(roundCurrency(amountYuan), 'f', -1, 64)
}

func paygPythonStringField(key, value string) paygPythonJSONField {
	return paygPythonJSONField{
		key: key,
		raw: strconv.QuoteToASCII(value),
	}
}

func paygPythonRawField(key, raw string) paygPythonJSONField {
	return paygPythonJSONField{
		key: key,
		raw: raw,
	}
}

func marshalPythonStyleJSONObject(fields ...paygPythonJSONField) string {
	var builder strings.Builder
	builder.Grow(len(fields) * 24)
	builder.WriteByte('{')
	for i, field := range fields {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(strconv.Quote(field.key))
		builder.WriteString(": ")
		builder.WriteString(field.raw)
	}
	builder.WriteByte('}')
	return builder.String()
}

func logPaygProviderFailure(action, payload string, meta *shouqianbaRequestMeta, resultCode, bizResult, providerErr, providerBizErr string) {
	httpStatus := 0
	responseBody := ""
	if meta != nil {
		httpStatus = meta.HTTPStatus
		responseBody = meta.Body
	}
	log.Printf(
		"[PAYG] shouqianba %s rejected: http_status=%d result_code=%s biz_result=%s err=%q biz_err=%q request=%s response=%s",
		action,
		httpStatus,
		strings.TrimSpace(resultCode),
		strings.TrimSpace(bizResult),
		strings.TrimSpace(providerErr),
		strings.TrimSpace(providerBizErr),
		truncateForLog(payload, 512),
		truncateForLog(responseBody, 1024),
	)
}

func newPaygProviderRejectedError(meta *shouqianbaRequestMeta, resultCode, bizResult, providerErr, providerBizErr string) error {
	metadata := map[string]string{}
	if meta != nil && meta.HTTPStatus > 0 {
		metadata["provider_http_status"] = strconv.Itoa(meta.HTTPStatus)
	}
	if trimmed := strings.TrimSpace(resultCode); trimmed != "" {
		metadata["provider_result_code"] = trimmed
	}
	if trimmed := strings.TrimSpace(bizResult); trimmed != "" {
		metadata["provider_biz_result_code"] = trimmed
	}
	combinedError := strings.TrimSpace(strings.Join(filterNonEmptyStrings(providerErr, providerBizErr), " | "))
	if combinedError != "" {
		metadata["provider_error"] = truncateForLog(combinedError, 200)
	}
	if len(metadata) == 0 {
		return ErrPaygProviderRejected
	}
	return ErrPaygProviderRejected.WithMetadata(metadata)
}

func filterNonEmptyStrings(values ...string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func truncateForLog(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}
