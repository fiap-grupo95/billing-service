package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
)

var ErrMissingMercadoPagoAccessToken = errors.New("missing MERCADOPAGO_ACCESS_TOKEN")
var ErrMercadoPagoGatewayNotConfigured = errors.New("mercado pago gateway not configured")

type MercadoPagoGateway struct {
	client   payment.Client
	mockMode bool
}

func NewMercadoPagoGateway(accessToken string) (*MercadoPagoGateway, error) {
	if isPaymentGatewayMockEnabled() {
		log.Printf("[payment][gateway] mock mode enabled")
		return &MercadoPagoGateway{mockMode: true}, nil
	}

	if accessToken == "" {
		log.Printf("[payment][gateway] missing MERCADOPAGO_ACCESS_TOKEN")
		return nil, ErrMissingMercadoPagoAccessToken
	}

	cfg, err := config.New(accessToken)
	if err != nil {
		log.Printf("[payment][gateway] failed creating sdk config err=%v", err)
		return nil, err
	}
	log.Printf("[payment][gateway] Mercado Pago client initialized")

	return &MercadoPagoGateway{client: payment.NewClient(cfg)}, nil
}

func (g *MercadoPagoGateway) CreatePayment(ctx context.Context, requestPayload json.RawMessage) (providerPaymentID string, providerStatus string, providerResponse json.RawMessage, err error) {
	if g != nil && g.mockMode {
		log.Printf("[payment][gateway] mock create start payload_len=%d", len(requestPayload))

		resp := map[string]any{}
		if len(requestPayload) > 0 && json.Valid(requestPayload) {
			if err := json.Unmarshal(requestPayload, &resp); err != nil {
				resp = map[string]any{"request_payload_raw": string(requestPayload)}
			}
		}

		id := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
		now := time.Now().UTC().Format(time.RFC3339Nano)
		resp["id"] = id
		resp["status"] = "approved"
		resp["status_detail"] = "accredited"
		if _, ok := resp["date_created"]; !ok {
			resp["date_created"] = now
		}
		if _, ok := resp["date_approved"]; !ok {
			resp["date_approved"] = now
		}

		b, err := json.Marshal(resp)
		if err != nil {
			log.Printf("[payment][gateway] mock response marshal failed err=%v", err)
			return "", "", nil, err
		}

		log.Printf("[payment][gateway] mock create success provider_payment_id=%s provider_status=approved", id)
		return id, "approved", b, nil
	}

	if g == nil || g.client == nil {
		log.Printf("[payment][gateway] gateway not configured")
		return "", "", nil, ErrMercadoPagoGatewayNotConfigured
	}
	log.Printf("[payment][gateway] create start payload_len=%d", len(requestPayload))

	var req payment.Request
	if err := json.Unmarshal(requestPayload, &req); err != nil {
		log.Printf("[payment][gateway] payload unmarshal failed err=%v", err)
		return "", "", nil, err
	}

	resp, err := g.client.Create(ctx, req)
	if err != nil {
		log.Printf("[payment][gateway] sdk create failed err=%v", err)
		return "", "", nil, err
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[payment][gateway] response marshal failed err=%v", err)
		return "", "", nil, err
	}
	log.Printf("[payment][gateway] create success provider_payment_id=%d provider_status=%s", resp.ID, resp.Status)

	return fmt.Sprintf("%d", resp.ID), resp.Status, b, nil
}

func isPaymentGatewayMockEnabled() bool {
	for _, key := range []string{"PAYMENT_GATEWAY_MOCK", "MERCADOPAGO_MOCK"} {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		switch v {
		case "1", "true", "yes", "on", "mock":
			return true
		}
	}
	return false
}
