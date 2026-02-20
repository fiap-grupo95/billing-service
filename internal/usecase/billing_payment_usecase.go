package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mecanica_xpto/internal/domain/entities"
	"mecanica_xpto/internal/usecase/interfaces"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrBillingPaymentNotFound         = errors.New("billing payment not found")
	ErrInvalidPaymentEstimateID       = errors.New("invalid estimate_id")
	ErrInvalidMPPayload               = errors.New("invalid mercado pago payload")
	ErrEstimateNotApproved            = errors.New("estimate not approved")
	ErrPaymentGatewayBadRequest       = errors.New("payment gateway bad request")
	ErrPaymentGatewayUnauthorized     = errors.New("payment gateway unauthorized")
	ErrPaymentGatewayInvalidUsers     = errors.New("payment gateway invalid users involved")
	ErrPaymentGatewayCustomerNotFound = errors.New("payment gateway customer not found")
)

// IBillingPaymentUseCase encapsulates the "create and process payment" behavior.
//
// Requested behavior:
//   - Create an item in the payment table and approve it as paid.

type IBillingPaymentUseCase interface {
	CreateAndApprove(ctx context.Context, estimateID string, mpPayload json.RawMessage) (entities.BillingPayment, error)
	GetByID(ctx context.Context, id string) (entities.BillingPayment, error)
	ListByEstimateID(ctx context.Context, estimateID string) ([]entities.BillingPayment, error)
}

type BillingPaymentUseCase struct {
	repo         interfaces.IBillingPaymentRepository
	estimateRepo interfaces.IEstimateRepository
	gateway      interfaces.IPaymentGateway
}

var _ IBillingPaymentUseCase = (*BillingPaymentUseCase)(nil)

func NewBillingPaymentUseCase(repo interfaces.IBillingPaymentRepository, estimateRepo interfaces.IEstimateRepository, gateway interfaces.IPaymentGateway) *BillingPaymentUseCase {
	return &BillingPaymentUseCase{repo: repo, estimateRepo: estimateRepo, gateway: gateway}
}

func (u *BillingPaymentUseCase) CreateAndApprove(ctx context.Context, estimateID string, mpPayload json.RawMessage) (entities.BillingPayment, error) {
	log.Printf("[payment][usecase] create-and-approve start raw_estimate_id=%q payload_len=%d", estimateID, len(mpPayload))
	mockMode := isPaymentGatewayMockEnabled()
	estimateID = strings.TrimSpace(estimateID)
	if estimateID == "" {
		log.Printf("[payment][usecase] invalid estimate_id (empty)")
		return entities.BillingPayment{}, ErrInvalidPaymentEstimateID
	}
	if len(mpPayload) == 0 {
		if mockMode {
			mpPayload = json.RawMessage("{}")
		} else {
			log.Printf("[payment][usecase] invalid payload (empty) estimate_id=%s", estimateID)
			return entities.BillingPayment{}, ErrInvalidMPPayload
		}
	}
	if !json.Valid(mpPayload) {
		if mockMode {
			mpPayload = json.RawMessage("{}")
		} else {
			log.Printf("[payment][usecase] invalid payload (not-json) estimate_id=%s", estimateID)
			return entities.BillingPayment{}, ErrInvalidMPPayload
		}
	}
	if u.gateway == nil {
		log.Printf("[payment][usecase] gateway not configured estimate_id=%s", estimateID)
		return entities.BillingPayment{}, errors.New("payment gateway not configured")
	}
	if u.estimateRepo == nil {
		log.Printf("[payment][usecase] estimate repository not configured estimate_id=%s", estimateID)
		return entities.BillingPayment{}, errors.New("estimate repository not configured")
	}

	log.Printf("[payment][usecase] loading estimate estimate_id=%s", estimateID)
	var err error
	est, err := u.estimateRepo.GetByID(ctx, estimateID)
	if err != nil {
		log.Printf("[payment][usecase] failed loading estimate estimate_id=%s err=%v", estimateID, err)
		return entities.BillingPayment{}, err
	}
	if est.ID == "" {
		log.Printf("[payment][usecase] estimate not found estimate_id=%s", estimateID)
		return entities.BillingPayment{}, ErrEstimateNotFound
	}
	if !mockMode && est.Status != entities.EstimateStatusAprovado {
		log.Printf("[payment][usecase] estimate not approved estimate_id=%s status=%s", estimateID, est.Status)
		return entities.BillingPayment{}, ErrEstimateNotApproved
	}
	log.Printf("[payment][usecase] estimate loaded estimate_id=%s status=%s price=%.2f", estimateID, est.Status, est.Price)

	// Ensure basic linkage with the estimate when the caller didn't provide it.
	// Mercado Pago uses external_reference to help reconcile events.
	var reqMap map[string]any
	if err := json.Unmarshal(mpPayload, &reqMap); err == nil {
		if !mockMode && !hasNonEmptyString(reqMap, "payment_method_id") {
			log.Printf("[payment][usecase] missing payment_method_id estimate_id=%s", estimateID)
			return entities.BillingPayment{}, ErrInvalidMPPayload
		}
		if !mockMode {
			normalizeSandboxPayerFromUserID(reqMap)
			ensurePayerDefaults(reqMap)
		}
		if !mockMode && !hasPayer(reqMap) {
			log.Printf("[payment][usecase] missing/invalid payer estimate_id=%s", estimateID)
			return entities.BillingPayment{}, ErrInvalidMPPayload
		}

		log.Printf("[payment][usecase] enriching payload estimate_id=%s", estimateID)
		if _, ok := reqMap["external_reference"]; !ok {
			reqMap["external_reference"] = estimateID
		}
		if _, ok := reqMap["description"]; !ok {
			reqMap["description"] = fmt.Sprintf("Estimate %s", estimateID)
		}

		// The source of truth for amount is the estimate in DB.
		reqMap["transaction_amount"] = est.Price
		if b, err := json.Marshal(reqMap); err == nil {
			mpPayload = b
			log.Printf("[payment][usecase] payload enriched estimate_id=%s payload_len=%d", estimateID, len(mpPayload))
		}
	} else {
		log.Printf("[payment][usecase] payload unmarshal failed estimate_id=%s err=%v", estimateID, err)
	}

	providerPaymentID := ""
	providerStatus := ""
	providerResp := json.RawMessage(nil)

	if mockMode {
		log.Printf("[payment][usecase] mock mode enabled; skipping external payment gateway estimate_id=%s", estimateID)
		providerPaymentID = strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
		providerStatus = "approved"
		now := time.Now().UTC().Format(time.RFC3339Nano)
		mockResp := map[string]any{}
		if len(mpPayload) > 0 && json.Valid(mpPayload) {
			_ = json.Unmarshal(mpPayload, &mockResp)
		}
		mockResp["id"] = providerPaymentID
		mockResp["status"] = "approved"
		mockResp["status_detail"] = "accredited"
		mockResp["date_created"] = now
		mockResp["date_approved"] = now
		if _, ok := mockResp["external_reference"]; !ok {
			mockResp["external_reference"] = estimateID
		}
		if _, ok := mockResp["transaction_amount"]; !ok {
			mockResp["transaction_amount"] = est.Price
		}
		b, mErr := json.Marshal(mockResp)
		if mErr != nil {
			return entities.BillingPayment{}, mErr
		}
		providerResp = b
	} else {
		log.Printf("[payment][usecase] calling payment gateway estimate_id=%s", estimateID)
		providerPaymentID, providerStatus, providerResp, err = u.gateway.CreatePayment(ctx, mpPayload)
		if err != nil {
			log.Printf("[payment][usecase] payment gateway failed estimate_id=%s err=%v", estimateID, err)
			if isGatewayCustomerNotFound(err) {
				return entities.BillingPayment{}, ErrPaymentGatewayCustomerNotFound
			}
			if isGatewayInvalidUsers(err) {
				return entities.BillingPayment{}, ErrPaymentGatewayInvalidUsers
			}
			if isGatewayUnauthorized(err) {
				return entities.BillingPayment{}, ErrPaymentGatewayUnauthorized
			}
			if isGatewayBadRequest(err) {
				return entities.BillingPayment{}, ErrPaymentGatewayBadRequest
			}
			return entities.BillingPayment{}, err
		}
	}
	log.Printf("[payment][usecase] payment gateway success estimate_id=%s provider_payment_id=%s provider_status=%s", estimateID, providerPaymentID, providerStatus)

	status := entities.PaymentStatusAprovado

	var parsed map[string]interface{}
	if err := json.Unmarshal(providerResp, &parsed); err != nil {
		log.Printf("[payment][usecase] provider response unmarshal failed estimate_id=%s err=%v", estimateID, err)
	}

	now := time.Now().UTC()
	p := entities.BillingPayment{
		ID:           providerPaymentID,
		EstimateID:   estimateID,
		Date:         now,
		Status:       status,
		MPPayloadRaw: providerResp,
		MPPayload:    parsed,
	}

	created, err := u.repo.Create(ctx, p)
	if err != nil {
		log.Printf("[payment][usecase] payment repository create failed estimate_id=%s payment_id=%s err=%v", estimateID, p.ID, err)
		return entities.BillingPayment{}, err
	}
	log.Printf("[payment][usecase] create-and-approve success estimate_id=%s payment_id=%s status=%s", estimateID, created.ID, created.Status)
	return created, nil
}

func hasNonEmptyString(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(s) != ""
}

func hasPayer(m map[string]any) bool {
	v, ok := m["payer"]
	if !ok {
		return false
	}
	payer, ok := v.(map[string]any)
	if !ok {
		return false
	}
	return hasNonEmptyString(payer, "email") || hasPayerID(payer)
}

func hasPayerID(payer map[string]any) bool {
	v, ok := payer["id"]
	if !ok || v == nil {
		return false
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	return s != "" && s != "<nil>"
}

func ensurePayerDefaults(m map[string]any) {
	v, ok := m["payer"]
	if !ok || v == nil {
		v = map[string]any{}
		m["payer"] = v
	}
	payer, ok := v.(map[string]any)
	if !ok {
		return
	}

	if _, ok := payer["type"]; !ok {
		payer["type"] = "customer"
	}

	// In sandbox, either payer.id or payer.email may be used.
	// Fill email only when both are missing.
	if !hasPayerID(payer) && !hasNonEmptyString(payer, "email") {
		if email := strings.TrimSpace(os.Getenv("MERCADOPAGO_TEST_PAYER_EMAIL")); email != "" {
			payer["email"] = email
		} else if strings.HasPrefix(strings.TrimSpace(os.Getenv("MERCADOPAGO_ACCESS_TOKEN")), "TEST-") {
			// Sandbox-safe fallback recommended by Mercado Pago examples.
			payer["email"] = "test_user_br@testuser.com"
		}
	}
}

func normalizeSandboxPayerFromUserID(m map[string]any) {
	v, ok := m["payer"]
	if !ok || v == nil {
		return
	}
	payer, ok := v.(map[string]any)
	if !ok {
		return
	}

	if !hasPayerID(payer) || hasNonEmptyString(payer, "email") {
		return
	}

	accessToken := strings.TrimSpace(os.Getenv("MERCADOPAGO_ACCESS_TOKEN"))
	if !strings.HasPrefix(accessToken, "TEST-") {
		return
	}

	configuredUserID := strings.TrimSpace(os.Getenv("MERCADOPAGO_TEST_PAYER_USER_ID"))
	configuredEmail := strings.TrimSpace(os.Getenv("MERCADOPAGO_TEST_PAYER_EMAIL"))
	if configuredUserID == "" || configuredEmail == "" {
		return
	}

	rawID := strings.TrimSpace(fmt.Sprintf("%v", payer["id"]))
	if rawID == "" || rawID == "<nil>" || rawID != configuredUserID {
		return
	}

	payer["email"] = configuredEmail
	delete(payer, "id")
	log.Printf("[payment][usecase] mapped sandbox payer user_id to payer.email")
}

func isPaymentGatewayMockEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PAYMENT_GATEWAY_MOCK")))
	switch v {
	case "1", "true", "yes", "on", "mock":
		return true
	}

	v = strings.ToLower(strings.TrimSpace(os.Getenv("MERCADOPAGO_MOCK")))
	switch v {
	case "1", "true", "yes", "on", "mock":
		return true
	}

	return false
}

func isGatewayBadRequest(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "\"error\":\"bad_request\"") || strings.Contains(msg, "\"status\":400")
}

func isGatewayUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "\"error\":\"unauthorized\"") || strings.Contains(msg, "\"status\":401")
}

func isGatewayInvalidUsers(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid users involved") || strings.Contains(msg, "\"code\":2034")
}

func isGatewayCustomerNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "customer not found") || strings.Contains(msg, "\"code\":2002")
}

func (u *BillingPaymentUseCase) GetByID(ctx context.Context, id string) (entities.BillingPayment, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return entities.BillingPayment{}, errors.New("invalid payment id")
	}

	p, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return entities.BillingPayment{}, err
	}
	if p.ID == "" {
		return entities.BillingPayment{}, ErrBillingPaymentNotFound
	}
	return p, nil
}

func (u *BillingPaymentUseCase) ListByEstimateID(ctx context.Context, estimateID string) ([]entities.BillingPayment, error) {
	estimateID = strings.TrimSpace(estimateID)
	if estimateID == "" {
		return nil, ErrInvalidPaymentEstimateID
	}
	return u.repo.ListByEstimateID(ctx, estimateID)
}
