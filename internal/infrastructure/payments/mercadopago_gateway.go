package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
)

var ErrMissingMercadoPagoAccessToken = errors.New("missing MERCADOPAGO_ACCESS_TOKEN")
var ErrMercadoPagoGatewayNotConfigured = errors.New("mercado pago gateway not configured")

type MercadoPagoGateway struct {
	client payment.Client
}

func NewMercadoPagoGateway(accessToken string) (*MercadoPagoGateway, error) {
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
