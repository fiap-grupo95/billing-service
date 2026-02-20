package routes

import (
	"mecanica_xpto/internal/adapter/http/handlers"

	"github.com/gin-gonic/gin"
)

const (
	PathEstimates = "/estimates"
)

func addBillingRoutes(rg *gin.RouterGroup, estimateHandler *handlers.EstimateHandler, paymentHandler *handlers.BillingPaymentHandler) {
	estimates := rg.Group(PathEstimates)
	{
		// Endpoints compatíveis com IBillingServiceRepository.
		estimates.POST("", estimateHandler.CreateEstimate)
		estimates.PATCH("/approve", estimateHandler.ApproveEstimate)
		estimates.PATCH("/reject", estimateHandler.RejectEstimate)
		estimates.PATCH("/cancel", estimateHandler.CancelEstimate)
	}

	payments := rg.Group(PathPayments)
	{
		// Endpoints compatíveis com IBillingServiceRepository.
		payments.POST("/:estimate_id", paymentHandler.CreatePaymentByEstimateID)
		payments.GET("/:estimate_id", paymentHandler.GetPaymentByEstimateID)
	}
}
