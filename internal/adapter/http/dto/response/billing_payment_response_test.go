package response

import (
	"encoding/json"
	"testing"
	"time"

	"mecanica_xpto/internal/domain/entities"
)

func TestFromBillingPayment(t *testing.T) {
	now := time.Now().UTC()
	payload := map[string]interface{}{"a": "b"}
	raw := json.RawMessage(`{"id":123}`)

	p := entities.BillingPayment{
		ID:           "pay-1",
		EstimateID:   "est-1",
		Date:         now,
		Status:       entities.PaymentStatusAprovado,
		MPPayloadRaw: raw,
		MPPayload:    payload,
	}

	res := FromBillingPayment(p)
	if res.ID != "pay-1" || res.PaymentID != "pay-1" {
		t.Fatalf("unexpected ids: %+v", res)
	}
	if res.EstimateID != "est-1" || res.Status != "aprovado" {
		t.Fatalf("unexpected fields: %+v", res)
	}
	if !res.Date.Equal(now) || !res.PaymentDate.Equal(now) {
		t.Fatalf("unexpected dates: %+v", res)
	}
	if res.MPPayloadRaw != string(raw) {
		t.Fatalf("unexpected raw payload: %s", res.MPPayloadRaw)
	}
	if res.MPPayload["a"] != "b" {
		t.Fatalf("unexpected parsed payload: %+v", res.MPPayload)
	}
}
