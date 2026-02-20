APP_SERVICE_NAME=app

.PHONY: init up up-localstack down logs test swag-generate

init:
	@if [ ! -f .env ]; then cp .env-example .env; fi
	docker compose --profile dynamodb-local up -d --build

up:
	docker compose --profile dynamodb-local up -d --build

up-localstack:
	docker compose --profile localstack up -d --build

down:
	docker compose down

logs:
	docker compose logs -f $(APP_SERVICE_NAME)

test:
	go test ./... -v

swag-generate:
	go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g cmd/api/main.go --output ./docs --parseDependency --parseInternal