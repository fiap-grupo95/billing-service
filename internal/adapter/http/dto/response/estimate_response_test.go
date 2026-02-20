package response

import (
	"testing"
	"time"

	"mecanica_xpto/internal/domain/entities"
)

func TestFromEstimate(t *testing.T) {
	now := time.Now().UTC()
	e := entities.Estimate{
		ID:        "est-1",
		OSID:      "os-1",
		Price:     99.9,
		Status:    entities.EstimateStatusAprovado,
		CreatedAt: now,
		UpdatedAt: now,
	}

	res := FromEstimate(e)
	if res.ID != "est-1" || res.EstimateID != "est-1" {
		t.Fatalf("unexpected ids: %+v", res)
	}
	if res.OSID != "os-1" || res.ServiceOrderID != "os-1" {
		t.Fatalf("unexpected os id fields: %+v", res)
	}
	if res.Price != 99.9 || res.Status != "aprovado" {
		t.Fatalf("unexpected mapped fields: %+v", res)
	}
	if !res.CreatedAt.Equal(now) || !res.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected dates: %+v", res)
	}
}
