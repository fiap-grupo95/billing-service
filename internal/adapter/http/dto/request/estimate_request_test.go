package request

import (
	"errors"
	"testing"
)

func TestEstimateRequest_ResolveOSIDAndEstimateID(t *testing.T) {
	r := EstimateRequest{ServiceOrderID: " os-123 "}
	if got := r.ResolveOSID(); got != "os-123" {
		t.Fatalf("expected os-123, got %q", got)
	}
	if got := r.ResolveEstimateID(); got != "os-123" {
		t.Fatalf("expected os-123, got %q", got)
	}

	r2 := EstimateRequest{ServiceOrderID: "   "}
	if got := r2.ResolveOSID(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := r2.ResolveEstimateID(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestEstimateRequest_ResolvePrice(t *testing.T) {
	r := EstimateRequest{
		Services: []ServiceRequest{{Price: 10}, {Price: 5}, {Price: -1}},
		PartsSupplies: []PartsSupplyRequest{
			{Price: 3, Quantity: 2},
			{Price: 4, Quantity: 0},
			{Price: -2, Quantity: 10},
		},
	}
	price, err := r.ResolvePrice()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 21 {
		t.Fatalf("expected 21, got %v", price)
	}

	r2 := EstimateRequest{}
	_, err = r2.ResolvePrice()
	if !errors.Is(err, ErrInvalidEstimateValue) {
		t.Fatalf("expected ErrInvalidEstimateValue, got %v", err)
	}
}
