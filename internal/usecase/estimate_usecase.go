package usecase

import (
	"context"
	"errors"
	"mecanica_xpto/internal/domain/entities"
	"mecanica_xpto/internal/usecase/interfaces"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEstimateNotFound      = errors.New("estimate not found")
	ErrEstimateAlreadyExists = errors.New("estimate already exists")
	ErrInvalidOSID           = errors.New("invalid os_id")
	ErrInvalidEstimateID     = errors.New("invalid estimate id")
	ErrInvalidEstimateVal    = errors.New("invalid estimate value")
)

// IEstimateUseCase exposes billing estimate operations.
//
// These operations directly map to the draw.io requirements:
//   - "Calcula Orçamento" => CalculateEstimate()
//   - PATCH /os/{id}/estimate (acao aprovar/rejeitar/cancelar) => UpdateStatusByOSAction()
//   - "Recalcula Orçamento Total" => UpdateEstimatePrice()

type IEstimateUseCase interface {
	CalculateEstimate(ctx context.Context, osID string, price float64) (entities.Estimate, error)
	ApproveByOSID(ctx context.Context, osID string) (entities.Estimate, error)
	RejectByOSID(ctx context.Context, osID string) (entities.Estimate, error)
	CancelByOSID(ctx context.Context, osID string) (entities.Estimate, error)
	UpdateEstimatePrice(ctx context.Context, estimateID string, newPrice float64) (entities.Estimate, error)
	GetByID(ctx context.Context, id string) (entities.Estimate, error)
	GetByOSID(ctx context.Context, osID string) (entities.Estimate, error)
}

type EstimateUseCase struct {
	repo interfaces.IEstimateRepository
}

var _ IEstimateUseCase = (*EstimateUseCase)(nil)

func NewEstimateUseCase(repo interfaces.IEstimateRepository) *EstimateUseCase {
	return &EstimateUseCase{repo: repo}
}

func (u *EstimateUseCase) CalculateEstimate(ctx context.Context, osID string, price float64) (entities.Estimate, error) {
	osID = strings.TrimSpace(osID)
	if osID == "" {
		return entities.Estimate{}, ErrInvalidOSID
	}
	if price <= 0 {
		return entities.Estimate{}, ErrInvalidEstimateVal
	}

	// Enforce: 1 estimate per OS.
	if existing, err := u.repo.GetByOSID(ctx, osID); err != nil {
		return entities.Estimate{}, err
	} else if existing.ID != "" {
		return entities.Estimate{}, ErrEstimateAlreadyExists
	}

	now := time.Now().UTC()
	e := entities.Estimate{
		ID:        uuid.NewString(),
		OSID:      osID,
		Price:     price,
		Status:    entities.EstimateStatusPendente,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return u.repo.Create(ctx, e)
}

func (u *EstimateUseCase) ApproveByOSID(ctx context.Context, osID string) (entities.Estimate, error) {
	return u.updateStatusByOSID(ctx, osID, entities.EstimateStatusAprovado)
}

func (u *EstimateUseCase) RejectByOSID(ctx context.Context, osID string) (entities.Estimate, error) {
	return u.updateStatusByOSID(ctx, osID, entities.EstimateStatusRejeitado)
}

func (u *EstimateUseCase) CancelByOSID(ctx context.Context, osID string) (entities.Estimate, error) {
	return u.updateStatusByOSID(ctx, osID, entities.EstimateStatusCancelado)
}

func (u *EstimateUseCase) updateStatusByOSID(ctx context.Context, osID string, status entities.EstimateStatus) (entities.Estimate, error) {
	osID = strings.TrimSpace(osID)
	if osID == "" {
		return entities.Estimate{}, ErrInvalidOSID
	}

	updated, err := u.repo.UpdateStatusByOSID(ctx, osID, status)
	if err != nil {
		return entities.Estimate{}, err
	}
	if updated.ID == "" {
		return entities.Estimate{}, ErrEstimateNotFound
	}
	return updated, nil
}

func (u *EstimateUseCase) UpdateEstimatePrice(ctx context.Context, estimateID string, newPrice float64) (entities.Estimate, error) {
	estimateID = strings.TrimSpace(estimateID)
	if estimateID == "" {
		return entities.Estimate{}, ErrInvalidEstimateID
	}
	if newPrice <= 0 {
		return entities.Estimate{}, ErrInvalidEstimateVal
	}

	updated, err := u.repo.UpdatePriceByID(ctx, estimateID, newPrice)
	if err != nil {
		return entities.Estimate{}, err
	}
	if updated.ID == "" {
		return entities.Estimate{}, ErrEstimateNotFound
	}
	return updated, nil
}

func (u *EstimateUseCase) GetByID(ctx context.Context, id string) (entities.Estimate, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return entities.Estimate{}, ErrInvalidEstimateID
	}

	e, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return entities.Estimate{}, err
	}
	if e.ID == "" {
		return entities.Estimate{}, ErrEstimateNotFound
	}
	return e, nil
}

func (u *EstimateUseCase) GetByOSID(ctx context.Context, osID string) (entities.Estimate, error) {
	osID = strings.TrimSpace(osID)
	if osID == "" {
		return entities.Estimate{}, ErrInvalidOSID
	}

	e, err := u.repo.GetByOSID(ctx, osID)
	if err != nil {
		return entities.Estimate{}, err
	}
	if e.ID == "" {
		return entities.Estimate{}, ErrEstimateNotFound
	}
	return e, nil
}
