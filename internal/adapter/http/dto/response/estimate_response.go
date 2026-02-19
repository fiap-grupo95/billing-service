package response

import (
	"mecanica_xpto/internal/domain/entities"
	"time"
)

type EstimateResponse struct {
	EstimateID     string    `json:"estimate_id"`
	ID             string    `json:"id"`
	ServiceOrderID string    `json:"service_order_id"`
	OSID           string    `json:"os_id"`
	Price          float64   `json:"price"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func FromEstimate(e entities.Estimate) EstimateResponse {
	return EstimateResponse{
		EstimateID:     e.ID,
		ID:             e.ID,
		ServiceOrderID: e.OSID,
		OSID:           e.OSID,
		Price:          e.Price,
		Status:         string(e.Status),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}
