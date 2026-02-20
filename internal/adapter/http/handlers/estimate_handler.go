package handlers

import (
	"context"
	"errors"
	request "mecanica_xpto/internal/adapter/http/dto/request"
	response "mecanica_xpto/internal/adapter/http/dto/response"
	"mecanica_xpto/internal/domain/entities"
	"mecanica_xpto/internal/usecase"
	"mecanica_xpto/pkg"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	errInvalidEstimatePayload = pkg.NewDomainErrorSimple("INVALID_ESTIMATE_INPUT", "Invalid estimate payload", http.StatusBadRequest)
)

// EstimateHandler handles HTTP requests for billing estimates.
//
// This handler implements only the Billing Service responsibilities from the draw.io.

type EstimateHandler struct {
	usecase usecase.IEstimateUseCase
}

func NewEstimateHandler(uc usecase.IEstimateUseCase) *EstimateHandler {
	return &EstimateHandler{usecase: uc}
}

// CreateEstimate handles integration-compatible estimate creation requests.
//
// It accepts the EstimateRequest payload used by os-service-api and translates
// it into the domain command expected by the use case.
func (h *EstimateHandler) CreateEstimate(c *gin.Context) {
	var payload request.EstimateRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(errInvalidEstimatePayload.HTTPStatus, errInvalidEstimatePayload.ToHTTPError())
		return
	}

	osID := payload.ResolveOSID()
	if osID == "" {
		c.JSON(http.StatusBadRequest, pkg.NewDomainErrorSimple("INVALID_REQUEST", "Invalid request", http.StatusBadRequest).ToHTTPError())
		return
	}

	price, err := payload.ResolvePrice()
	if err != nil {
		c.JSON(errInvalidEstimatePayload.HTTPStatus, errInvalidEstimatePayload.ToHTTPError())
		return
	}

	estimate, err := h.usecase.CalculateEstimate(c.Request.Context(), osID, price)
	if err != nil {
		appErr := mapEstimateError(err)
		c.JSON(appErr.HTTPStatus, appErr.ToHTTPError())
		return
	}

	c.JSON(http.StatusCreated, response.FromEstimate(estimate))
}

func (h *EstimateHandler) ApproveEstimate(c *gin.Context) {
	h.patchEstimateStatusByRequest(c, h.usecase.ApproveByOSID)
}

func (h *EstimateHandler) RejectEstimate(c *gin.Context) {
	h.patchEstimateStatusByRequest(c, h.usecase.RejectByOSID)
}

func (h *EstimateHandler) CancelEstimate(c *gin.Context) {
	h.patchEstimateStatusByRequest(c, h.usecase.CancelByOSID)
}

func (h *EstimateHandler) patchEstimateStatusByRequest(
	c *gin.Context,
	updater func(ctx context.Context, osID string) (entities.Estimate, error),
) {
	var payload request.EstimateRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(errInvalidEstimatePayload.HTTPStatus, errInvalidEstimatePayload.ToHTTPError())
		return
	}

	osID := payload.ResolveOSID()
	if osID == "" {
		c.JSON(http.StatusBadRequest, pkg.NewDomainErrorSimple("INVALID_REQUEST", "Invalid request", http.StatusBadRequest).ToHTTPError())
		return
	}

	estimate, err := updater(c.Request.Context(), osID)
	if err != nil {
		appErr := mapEstimateError(err)
		c.JSON(appErr.HTTPStatus, appErr.ToHTTPError())
		return
	}

	c.JSON(http.StatusOK, response.FromEstimate(estimate))
}

func mapEstimateError(err error) *pkg.AppError {
	switch {
	case errors.Is(err, usecase.ErrInvalidOSID), errors.Is(err, usecase.ErrInvalidEstimateID), errors.Is(err, usecase.ErrInvalidEstimateVal):
		return pkg.NewDomainErrorSimple("INVALID_REQUEST", "Invalid request", http.StatusBadRequest)
	case errors.Is(err, usecase.ErrEstimateAlreadyExists):
		return pkg.NewDomainErrorSimple("ESTIMATE_ALREADY_EXISTS", "Estimate already exists for this OS", http.StatusConflict)
	case errors.Is(err, usecase.ErrEstimateNotFound):
		return pkg.NewDomainErrorSimple("ESTIMATE_NOT_FOUND", "Estimate not found", http.StatusNotFound)
	default:
		return pkg.NewDomainError("INTERNAL_ERROR", "An internal error occurred", err, http.StatusInternalServerError)
	}
}
