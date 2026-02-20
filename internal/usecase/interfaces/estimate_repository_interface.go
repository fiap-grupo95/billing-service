package interfaces

import (
	"context"
	"mecanica_xpto/internal/domain/entities"
)

// IEstimateRepository abstracts DynamoDB persistence for Estimate.
//
// The billing-service must be able to:
//   - create an estimate when OS Service requests calculation
//   - update estimate status by OS ID (approve/reject/cancel)
//   - update estimate value by estimate ID (recalculation with additional repairs)

type IEstimateRepository interface {
	Create(ctx context.Context, e entities.Estimate) (entities.Estimate, error)
	GetByID(ctx context.Context, id string) (entities.Estimate, error)
	GetByOSID(ctx context.Context, osID string) (entities.Estimate, error)
	UpdateStatusByOSID(ctx context.Context, osID string, status entities.EstimateStatus) (entities.Estimate, error)
	UpdatePriceByID(ctx context.Context, id string, newPrice float64) (entities.Estimate, error)
}
