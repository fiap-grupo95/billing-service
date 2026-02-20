package handlers

import (
	"encoding/json"
	"errors"
	"log"
	response "mecanica_xpto/internal/adapter/http/dto/response"
	"mecanica_xpto/internal/usecase"
	"mecanica_xpto/pkg"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// BillingPaymentHandler handles HTTP requests for Billing payments.

type BillingPaymentHandler struct {
	usecase usecase.IBillingPaymentUseCase
}

func NewBillingPaymentHandler(uc usecase.IBillingPaymentUseCase) *BillingPaymentHandler {
	return &BillingPaymentHandler{usecase: uc}
}

// CreatePaymentByEstimateID creates/approves a payment using estimate_id in path.
func (h *BillingPaymentHandler) CreatePaymentByEstimateID(c *gin.Context) {
	estimateID := c.Param("estimate_id")
	log.Printf("[payment][handler] create start estimate_id=%s", estimateID)
	mockMode := isPaymentGatewayMockEnabled()
	mpPayload, err := readMPPayload(c)
	if err != nil {
		if mockMode {
			log.Printf("[payment][handler] payload invalid in mock mode; fallback to empty payload estimate_id=%s err=%v", estimateID, err)
			mpPayload = json.RawMessage("{}")
		} else {
			log.Printf("[payment][handler] invalid payload estimate_id=%s err=%v", estimateID, err)
			appErr := pkg.NewDomainErrorSimple("INVALID_REQUEST", "Invalid request", http.StatusBadRequest)
			c.JSON(appErr.HTTPStatus, appErr.ToHTTPError())
			return
		}
	}

	created, err := h.usecase.CreateAndApprove(c.Request.Context(), estimateID, mpPayload)
	if err != nil {
		log.Printf("[payment][handler] create failed estimate_id=%s err=%v", estimateID, err)
		appErr := mapBillingPaymentError(err)
		c.JSON(appErr.HTTPStatus, appErr.ToHTTPError())
		return
	}
	log.Printf("[payment][handler] create success estimate_id=%s payment_id=%s status=%s", estimateID, created.ID, created.Status)

	c.JSON(http.StatusOK, response.FromBillingPayment(created))
}

// GetPaymentByEstimateID returns the latest payment for an estimate.
func (h *BillingPaymentHandler) GetPaymentByEstimateID(c *gin.Context) {
	estimateID := c.Param("estimate_id")
	log.Printf("[payment][handler] get-by-estimate start estimate_id=%s", estimateID)

	payments, err := h.usecase.ListByEstimateID(c.Request.Context(), estimateID)
	if err != nil {
		log.Printf("[payment][handler] get-by-estimate failed estimate_id=%s err=%v", estimateID, err)
		appErr := mapBillingPaymentError(err)
		c.JSON(appErr.HTTPStatus, appErr.ToHTTPError())
		return
	}

	if len(payments) == 0 {
		log.Printf("[payment][handler] get-by-estimate not-found estimate_id=%s", estimateID)
		appErr := pkg.NewDomainErrorSimple("PAYMENT_NOT_FOUND", "Payment not found", http.StatusNotFound)
		c.JSON(appErr.HTTPStatus, appErr.ToHTTPError())
		return
	}

	latest := payments[0]
	for _, p := range payments[1:] {
		if p.Date.After(latest.Date) {
			latest = p
		}
	}
	log.Printf("[payment][handler] get-by-estimate success estimate_id=%s payment_id=%s status=%s", estimateID, latest.ID, latest.Status)

	c.JSON(http.StatusOK, response.FromBillingPayment(latest))
}

func readMPPayload(c *gin.Context) (json.RawMessage, error) {
	raw, err := c.GetRawData()
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return json.RawMessage("{}"), nil
	}
	if !json.Valid(raw) {
		return nil, errors.New("request body is not valid json")
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err == nil {
		if wrapped, ok := envelope["mp_payload"]; ok {
			if len(strings.TrimSpace(string(wrapped))) == 0 || strings.TrimSpace(string(wrapped)) == "null" {
				return nil, errors.New("mp_payload cannot be empty")
			}
			return wrapped, nil
		}
	}

	return json.RawMessage(raw), nil
}

func mapBillingPaymentError(err error) *pkg.AppError {
	switch {
	case errors.Is(err, usecase.ErrInvalidPaymentEstimateID), errors.Is(err, usecase.ErrInvalidMPPayload), errors.Is(err, usecase.ErrPaymentGatewayBadRequest):
		return pkg.NewDomainErrorSimple("INVALID_REQUEST", "Invalid request", http.StatusBadRequest)
	case errors.Is(err, usecase.ErrPaymentGatewayCustomerNotFound):
		return pkg.NewDomainErrorSimple("PAYMENT_PROVIDER_CUSTOMER_NOT_FOUND", "Payer not found for this Mercado Pago test context", http.StatusBadRequest)
	case errors.Is(err, usecase.ErrPaymentGatewayInvalidUsers):
		return pkg.NewDomainErrorSimple("PAYMENT_PROVIDER_INVALID_USERS", "Invalid users involved between seller token and payer test user", http.StatusBadRequest)
	case errors.Is(err, usecase.ErrPaymentGatewayUnauthorized):
		return pkg.NewDomainErrorSimple("PAYMENT_PROVIDER_UNAUTHORIZED", "Payment provider unauthorized", http.StatusUnauthorized)
	case errors.Is(err, usecase.ErrEstimateNotFound):
		return pkg.NewDomainErrorSimple("ESTIMATE_NOT_FOUND", "Estimate not found", http.StatusNotFound)
	case errors.Is(err, usecase.ErrEstimateNotApproved):
		return pkg.NewDomainErrorSimple("ESTIMATE_NOT_APPROVED", "Estimate not approved", http.StatusConflict)
	case errors.Is(err, usecase.ErrBillingPaymentNotFound):
		return pkg.NewDomainErrorSimple("PAYMENT_NOT_FOUND", "Payment not found", http.StatusNotFound)
	default:
		return pkg.NewDomainError("INTERNAL_ERROR", "An internal error occurred", err, http.StatusInternalServerError)
	}
}

func isPaymentGatewayMockEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PAYMENT_GATEWAY_MOCK")))
	switch v {
	case "1", "true", "yes", "on", "mock":
		return true
	}

	v = strings.ToLower(strings.TrimSpace(os.Getenv("MERCADOPAGO_MOCK")))
	switch v {
	case "1", "true", "yes", "on", "mock":
		return true
	}

	return false
}
