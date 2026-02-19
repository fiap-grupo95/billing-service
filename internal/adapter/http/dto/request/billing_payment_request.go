package request

import "encoding/json"

// BillingPaymentCreateRequest is the payload for the "cria e processa pagamento" route.
//
// `mp_payload` is stored as-is (raw JSON) to support varying Mercado Pago schemas.

type BillingPaymentCreateRequest struct {
	MPPayload json.RawMessage `json:"mp_payload"`
}
