package entities

import "time"

// EstimateStatus represents the lifecycle of an estimate (orçamento).
//
// Domain notes:
//   - The billing-service is the source of truth for estimate/payment state.
//   - Status transitions are driven by the OS Service actions described in the draw.io.
//
//go:generate stringer -type=EstimateStatus

type EstimateStatus string

const (
	EstimateStatusPendente  EstimateStatus = "pendente"
	EstimateStatusAprovado  EstimateStatus = "aprovado"
	EstimateStatusRejeitado EstimateStatus = "rejeitado"
	EstimateStatusCancelado EstimateStatus = "cancelado"
)

// Estimate is the billing estimate (orçamento) persisted in DynamoDB.
//
// Storage model (DynamoDB):
//   - PK: id
//   - GSI1 (os_id-index): os_id
//
// Monetary representation:
//   - Price represents the calculated estimate total.
//
type Estimate struct {
	ID        string         `json:"id"`
	OSID      string         `json:"os_id"`
	Price     float64        `json:"price"`
	Status    EstimateStatus `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
