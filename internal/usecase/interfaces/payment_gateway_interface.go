package interfaces

import (
	"context"
	"encoding/json"
)

// IPaymentGateway abstracts external payment providers (e.g. Mercado Pago).
//
// The billing-service uses it to create/process a payment and persist the provider
// response payload for traceability.
type IPaymentGateway interface {
	CreatePayment(ctx context.Context, requestPayload json.RawMessage) (providerPaymentID string, providerStatus string, providerResponse json.RawMessage, err error)
}
