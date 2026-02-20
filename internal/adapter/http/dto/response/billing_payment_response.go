package response

import (
	"mecanica_xpto/internal/domain/entities"
	"time"
)

type BillingPaymentResponse struct {
	PaymentID   string    `json:"payment_id"`
	ID          string    `json:"id"`
	EstimateID  string    `json:"estimate_id"`
	PaymentDate time.Time `json:"payment_date"`
	Date        time.Time `json:"date"`
	Status      string    `json:"status"`

	MPPayloadRaw string                 `json:"mp_payload_raw,omitempty"`
	MPPayload    map[string]interface{} `json:"mp_payload,omitempty"`
}

func FromBillingPayment(p entities.BillingPayment) BillingPaymentResponse {
	return BillingPaymentResponse{
		PaymentID:    p.ID,
		ID:           p.ID,
		EstimateID:   p.EstimateID,
		PaymentDate:  p.Date,
		Date:         p.Date,
		Status:       string(p.Status),
		MPPayloadRaw: string(p.MPPayloadRaw),
		MPPayload:    p.MPPayload,
	}
}
