package routes

import (
	"log"
	_ "mecanica_xpto/docs" // This will be auto-generated
	"mecanica_xpto/internal/adapter/http/handlers"
	repository2 "mecanica_xpto/internal/adapter/persistence/repository"
	"mecanica_xpto/internal/infrastructure/database"
	"mecanica_xpto/internal/infrastructure/payments"
	"mecanica_xpto/internal/usecase"
	"mecanica_xpto/internal/usecase/interfaces"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var router = gin.Default()

const PORT = 8080

// Run will start the server
func Run() {
	setMiddlewares()

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	getRoutes()

	err := router.Run(":" + strconv.Itoa(PORT))
	if err != nil {
		log.Fatalf("Failed to startup the application: %v", err.Error())
	}
}

func getRoutes() {
	ddb := database.ConnectDynamoDB()

	estimateRepo := repository2.NewEstimateDynamoRepository(ddb)
	paymentRepo := repository2.NewBillingPaymentDynamoRepository(ddb)

	estimateUseCase := usecase.NewEstimateUseCase(estimateRepo)

	// DEBUG ONLY: explicit credential print requested by user.
	log.Printf("[debug][mp] MERCADOPAGO_PUBLIC_KEY=%s", os.Getenv("MERCADOPAGO_PUBLIC_KEY"))
	log.Printf("[debug][mp] MERCADOPAGO_ACCESS_TOKEN=%s", os.Getenv("MERCADOPAGO_ACCESS_TOKEN"))

	var paymentGateway interfaces.IPaymentGateway
	mpGateway, err := payments.NewMercadoPagoGateway(os.Getenv("MERCADOPAGO_ACCESS_TOKEN"))
	if err != nil {
		log.Printf("Mercado Pago gateway not configured: %v", err)
	} else {
		paymentGateway = mpGateway
	}

	paymentUseCase := usecase.NewBillingPaymentUseCase(paymentRepo, estimateRepo, paymentGateway)

	estimateHandler := handlers.NewEstimateHandler(estimateUseCase)
	billingPaymentHandler := handlers.NewBillingPaymentHandler(paymentUseCase)

	// Rotas publicas
	v1 := router.Group("/v1")
	addPingRoutes(v1)
	addBillingRoutes(v1, estimateHandler, billingPaymentHandler)
}

func setMiddlewares() {
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf("Recovered from panic: %v", recovered)
		c.AbortWithStatus(500)
	}))
}
