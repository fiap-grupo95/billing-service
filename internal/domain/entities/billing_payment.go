package entities

import (
	"encoding/json"
	"time"
)

// PaymentStatus represents the payment processing outcome.
//
// In the requested scope we only need to create/process and persist an approved payment.
// The type supports a denied status for completeness.

type PaymentStatus string

const (
	PaymentStatusPendente PaymentStatus = "pendente"
	PaymentStatusAprovado PaymentStatus = "aprovado"
	PaymentStatusNegado   PaymentStatus = "negado"
)

// BillingPayment is the payment entity persisted by the billing-service.
//
// Storage model (DynamoDB):
//   - PK: id
//   - GSI1 (estimate_id-index): estimate_id
//
// MercadoPago payload:
//   - MPPayloadRaw keeps the original body (JSON) for traceability/audit.
//   - MPPayload is an optional parsed representation, useful for querying/debugging.
//     (We persist both because different MP integrations may vary in schema.)

type BillingPayment struct {
	ID         string        `json:"id"`
	EstimateID string        `json:"estimate_id"`
	Date       time.Time     `json:"date"`
	Status     PaymentStatus `json:"status"`

	MPPayloadRaw json.RawMessage        `json:"mp_payload_raw,omitempty"`
	MPPayload    map[string]interface{} `json:"mp_payload,omitempty"`
}
