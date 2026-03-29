package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	subject := fmt.Sprintf("%s PAYG充值 ¥%.2f", s.settingService.GetSiteName(ctx), amountYuan)
	qrCode, sn, err := s.shouqianbaPrecreate(ctx, cfg, clientSN, amountCent, amountYuan, payway, subject, userID)
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
	if order.Status == PaygOrderStatusPaid {
		return order, nil
	}
	return s.syncOrderByIdentifiers(ctx, order.SN, order.ClientSN)
}

func (s *PaygService) HandleCallback(ctx context.Context, sn, clientSN string) (*PaygOrder, error) {
	return s.syncOrderByIdentifiers(ctx, strings.TrimSpace(sn), strings.TrimSpace(clientSN))
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
	amountYuan float64,
	payway string,
	subject string,
	userID int64,
) (qrCode string, sn string, err error) {
	reflectPayload, _ := json.Marshal(map[string]any{
		"user_id":     userID,
		"amount_yuan": roundCurrency(amountYuan),
	})
	body := map[string]any{
		"terminal_sn":  cfg.TerminalSN,
		"client_sn":    clientSN,
		"total_amount": strconv.FormatInt(amountCent, 10),
		"payway":       payway,
		"subject":      subject,
		"operator":     "system",
		"reflect":      string(reflectPayload),
	}

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
	if err := s.shouqianbaRequest(ctx, cfg, "/upay/v2/precreate", body, &resp); err != nil {
		return "", "", err
	}
	if resp.ResultCode != "200" || resp.BizResponse.ResultCode != "PRECREATE_SUCCESS" {
		return "", "", fmt.Errorf("payg precreate failed: result_code=%s biz_result=%s err=%s %s", resp.ResultCode, resp.BizResponse.ResultCode, resp.Error, resp.BizResponse.Error)
	}
	return strings.TrimSpace(resp.BizResponse.Data.QRCode), strings.TrimSpace(resp.BizResponse.Data.SN), nil
}

func (s *PaygService) shouqianbaQuery(ctx context.Context, cfg *PaygSettings, sn, clientSN string) (*PaygProviderOrderStatus, error) {
	body := map[string]any{
		"terminal_sn": cfg.TerminalSN,
	}
	if strings.TrimSpace(sn) != "" {
		body["sn"] = strings.TrimSpace(sn)
	} else if strings.TrimSpace(clientSN) != "" {
		body["client_sn"] = strings.TrimSpace(clientSN)
	}

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
	if err := s.shouqianbaRequest(ctx, cfg, "/upay/v2/query", body, &resp); err != nil {
		return nil, err
	}
	if resp.ResultCode != "200" {
		return nil, fmt.Errorf("payg query failed: result_code=%s err=%s", resp.ResultCode, resp.Error)
	}
	if isPaygQueryBizFailure(resp.BizResponse.ResultCode) {
		return nil, fmt.Errorf(
			"payg query failed: result_code=%s biz_result=%s err=%s %s",
			resp.ResultCode,
			resp.BizResponse.ResultCode,
			resp.Error,
			resp.BizResponse.Error,
		)
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

func (s *PaygService) shouqianbaRequest(ctx context.Context, cfg *PaygSettings, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal shouqianba request: %w", err)
	}

	sum := md5.Sum(append(payload, []byte(cfg.TerminalKey)...))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shouqianbaAPIBase+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create shouqianba request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.TerminalSN+" "+hex.EncodeToString(sum[:]))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send shouqianba request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read shouqianba response: %w", err)
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode shouqianba response: %w", err)
	}
	return nil
}

func generatePaygClientSN() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("PG%d%s", time.Now().UnixMilli(), hex.EncodeToString(b)), nil
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
