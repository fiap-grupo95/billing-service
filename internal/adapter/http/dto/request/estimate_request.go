package request

import (
	"errors"
	"strings"
)

var (
	ErrInvalidEstimateValue = errors.New("invalid estimate value")
)

type PartsSupplyRequest struct {
	ID          string  `json:"id"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Quantity    int     `json:"quantity" binding:"required"`
}

type ServiceRequest struct {
	ID          string  `json:"id"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
}

// EstimateRequest is an integration-facing payload accepted by compatibility
// endpoints used by os-service-api.
type EstimateRequest struct {
	AdditionalRepairID string               `json:"additional_repair_id"`
	ServiceOrderID     string               `json:"service_order_id" binding:"required"`
	Services           []ServiceRequest     `json:"services"`
	PartsSupplies      []PartsSupplyRequest `json:"parts_supplies"`
}

func (r EstimateRequest) ResolveOSID() string {
	if v := strings.TrimSpace(r.ServiceOrderID); v != "" {
		return v
	}
	return ""
}

func (r EstimateRequest) ResolveEstimateID() string {
	if v := strings.TrimSpace(r.ServiceOrderID); v != "" {
		return v
	}
	return ""
}

func (r EstimateRequest) ResolvePrice() (float64, error) {
	totalFromItems := 0.0
	for _, s := range r.Services {
		if s.Price > 0 {
			totalFromItems += s.Price
		}
	}
	for _, p := range r.PartsSupplies {
		if p.Price > 0 && p.Quantity > 0 {
			totalFromItems += p.Price * float64(p.Quantity)
		}
	}
	if totalFromItems > 0 {
		return totalFromItems, nil
	}

	return 0, ErrInvalidEstimateValue
}
