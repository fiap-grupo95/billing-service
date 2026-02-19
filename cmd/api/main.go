package main

import (
	_ "mecanica_xpto/docs"
	"mecanica_xpto/internal/adapter/http/routes"

	_ "github.com/joho/godotenv/autoload"
)

// @title           Billing Service API
// @version         1.0
// @description     Billing Service (estimates + payments) backed by DynamoDB.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080

// @BasePath  /v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	routes.Run()
}
