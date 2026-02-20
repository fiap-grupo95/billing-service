package interfaces

import (
	"context"
	"mecanica_xpto/internal/domain/entities"
)

// IBillingPaymentRepository abstracts DynamoDB persistence for BillingPayment.

type IBillingPaymentRepository interface {
	Create(ctx context.Context, p entities.BillingPayment) (entities.BillingPayment, error)
	GetByID(ctx context.Context, id string) (entities.BillingPayment, error)
	ListByEstimateID(ctx context.Context, estimateID string) ([]entities.BillingPayment, error)
}
