#!/usr/bin/env sh
set -eu

# Start only infra dependencies used by the app in VS Code debug mode.
# It does NOT start the billing-service app container.

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "${SCRIPT_DIR}/.." && pwd)"

cd "${ROOT_DIR}"

echo "[infra] Starting LocalStack + DynamoDB Local + init jobs..."
docker compose \
  --profile localstack \
  --profile dynamodb-local \
  up -d localstack localstack-init dynamodb dynamodb-init

echo "[infra] Current containers:"
docker compose ps

echo ""
echo "[infra] Endpoints available:"
echo "  - LocalStack (DynamoDB): http://localhost:4566"
echo "  - DynamoDB Local:        http://localhost:8000"
echo ""
echo "[app debug] Use one endpoint in VS Code debug env:"
echo "  DYNAMODB_ENDPOINT=http://localhost:4566   # LocalStack"
echo "  or"
echo "  DYNAMODB_ENDPOINT=http://localhost:8000   # DynamoDB Local"
