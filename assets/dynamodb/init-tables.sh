#!/bin/sh
set -eu

# Avoid AWS CLI trying IMDS inside containers.
export AWS_EC2_METADATA_DISABLED=true

ENDPOINT_URL="${DYNAMODB_ENDPOINT_URL:-http://dynamodb:8000}"
REGION="${AWS_REGION:-us-east-1}"

ESTIMATES_TABLE="${ESTIMATES_TABLE:-estimates}"
PAYMENTS_TABLE="${PAYMENTS_TABLE:-payments}"

wait_for_dynamo() {
  echo "Waiting for DynamoDB Local at ${ENDPOINT_URL}..."
  # DynamoDB Local responds to ListTables when ready.
  until aws dynamodb list-tables --endpoint-url "${ENDPOINT_URL}" --region "${REGION}" --no-cli-pager >/dev/null 2>&1; do
    sleep 1
  done
}

now_rfc3339nano() {
  # GNU date supports %N (nanoseconds). If not available, fallback to seconds.
  date -u +"%Y-%m-%dT%H:%M:%S.%NZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ"
}

create_table_if_missing() {
  TABLE="$1"
  if aws dynamodb describe-table --table-name "${TABLE}" --endpoint-url "${ENDPOINT_URL}" --region "${REGION}" >/dev/null 2>&1; then
    echo "Table ${TABLE} already exists"
    return 0
  fi
  shift
  echo "Creating table ${TABLE}..."
  aws dynamodb create-table --table-name "${TABLE}" --endpoint-url "${ENDPOINT_URL}" --region "${REGION}" "$@" >/dev/null
  echo "Created table ${TABLE}"
}

wait_for_dynamo

create_table_if_missing "${ESTIMATES_TABLE}" \
  --attribute-definitions AttributeName=id,AttributeType=S \
  --key-schema AttributeName=id,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST

create_table_if_missing "${PAYMENTS_TABLE}" \
  --attribute-definitions \
    AttributeName=id,AttributeType=S \
    AttributeName=estimate_id,AttributeType=S \
  --key-schema AttributeName=id,KeyType=HASH \
  --global-secondary-indexes \
    "IndexName=estimate_id-index,KeySchema=[{AttributeName=estimate_id,KeyType=HASH}],Projection={ProjectionType=ALL}" \
  --billing-mode PAY_PER_REQUEST

echo "DynamoDB tables ready."

# --- Seed demo data (1 record per table) ---

SEED_OS_ID="${SEED_OS_ID:-os_demo_1}"
SEED_ESTIMATE_PRICE="${SEED_ESTIMATE_PRICE:-100.0}"
SEED_ESTIMATE_VALUE_CENTS="${SEED_ESTIMATE_VALUE_CENTS:-10000}"
SEED_PAYMENT_ID="${SEED_PAYMENT_ID:-pay_demo_1}"

seed_estimate_if_missing() {
  ID="$1"
  NOW="$2"

  existing="$(aws dynamodb get-item \
    --table-name "${ESTIMATES_TABLE}" \
    --endpoint-url "${ENDPOINT_URL}" \
    --region "${REGION}" \
    --no-cli-pager \
    --key "{\"id\":{\"S\":\"${ID}\"}}" \
    --query 'Item.id.S' \
    --output text 2>/dev/null || true)"

  if [ "${existing}" != "None" ] && [ -n "${existing}" ]; then
    echo "Seed estimate already exists (id=${ID})"
    return 0
  fi

  echo "Seeding estimate (id=${ID})..."
  aws dynamodb put-item \
    --table-name "${ESTIMATES_TABLE}" \
    --endpoint-url "${ENDPOINT_URL}" \
    --region "${REGION}" \
    --no-cli-pager \
    --item "{\
      \"id\":{\"S\":\"${ID}\"},\
      \"os_id\":{\"S\":\"${ID}\"},\
      \"price\":{\"N\":\"${SEED_ESTIMATE_PRICE}\"},\
      \"status\":{\"S\":\"pendente\"},\
      \"created_at\":{\"S\":\"${NOW}\"},\
      \"updated_at\":{\"S\":\"${NOW}\"}\
    }" >/dev/null

  echo "Seeded estimate (id=${ID})"
}

seed_payment_if_missing() {
  PAY_ID="$1"
  EST_ID="$2"
  NOW="$3"

  existing="$(aws dynamodb get-item \
    --table-name "${PAYMENTS_TABLE}" \
    --endpoint-url "${ENDPOINT_URL}" \
    --region "${REGION}" \
    --no-cli-pager \
    --key "{\"id\":{\"S\":\"${PAY_ID}\"}}" \
    --query 'Item.id.S' \
    --output text 2>/dev/null || true)"

  if [ "${existing}" != "None" ] && [ -n "${existing}" ]; then
    echo "Seed payment already exists (id=${PAY_ID})"
    return 0
  fi

  echo "Seeding payment (id=${PAY_ID})..."

  # We seed both representations used by the Go entity/repository:
  # - mp_payload_raw: raw JSON string
  # - mp_payload: map representation
  aws dynamodb put-item \
    --table-name "${PAYMENTS_TABLE}" \
    --endpoint-url "${ENDPOINT_URL}" \
    --region "${REGION}" \
    --no-cli-pager \
    --item "{\
      \"id\":{\"S\":\"${PAY_ID}\"},\
      \"estimate_id\":{\"S\":\"${EST_ID}\"},\
      \"date\":{\"S\":\"${NOW}\"},\
      \"status\":{\"S\":\"aprovado\"},\
      \"mp_payload_raw\":{\"S\":\"{\\\"provider\\\":\\\"mercadopago\\\",\\\"transaction_id\\\":\\\"tx_demo_1\\\",\\\"amount_cents\\\":${SEED_ESTIMATE_VALUE_CENTS}}\"},\
      \"mp_payload\":{\"M\":{\
        \"provider\":{\"S\":\"mercadopago\"},\
        \"transaction_id\":{\"S\":\"tx_demo_1\"},\
        \"amount_cents\":{\"N\":\"${SEED_ESTIMATE_VALUE_CENTS}\"}\
      }}\
    }" >/dev/null

  echo "Seeded payment (id=${PAY_ID})"
}

NOW="$(now_rfc3339nano)"
seed_estimate_if_missing "${SEED_OS_ID}" "${NOW}"
seed_payment_if_missing "${SEED_PAYMENT_ID}" "${SEED_OS_ID}" "${NOW}"
