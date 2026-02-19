
# billing-service

## Objetivo

Este repositório foi refatorado para atuar como um **Billing Service**, responsável apenas por:

- **Orçamento (Estimate)**
- **Pagamento (Payment)**

Conforme o draw.io (abas **fluxo-os** e **reparos-adicionais**), este serviço implementa exclusivamente as operações de billing.

## Stack

- Go (Golang) + Gin
- DynamoDB (local via DynamoDB Local)
- (Opcional) LocalStack (DynamoDB)
- Swagger (swag)

## Tabelas (DynamoDB)

### estimates (orçamento)

- `id` (PK) *(string)* — **usa o `os_id` como id** (1 orçamento por OS)
- `os_id` *(string)*
- `value_cents` *(number)*
- `status` *(string)*: `pendente` | `aprovado` | `rejeitado` | `cancelado`
- `created_at` *(string RFC3339)*
- `updated_at` *(string RFC3339)*

### payments (pagamento)

- `id` (PK) *(string)*
- `estimate_id` *(string)* — GSI `estimate_id-index`
- `date` *(string RFC3339)*
- `status` *(string)*: `pendente` | `aprovado` | `negado`
- `mp_payload_raw` *(string JSON)*
- `mp_payload` *(map, opcional)*

## Rotas implementadas (Billing Service)

Base path: `/v1`

Este serviço expõe apenas os endpoints necessários para o contrato de integração com `IBillingServiceRepository`.

## Compatibilidade com `os-service-api` (`billing_service_interface.go`)

Além das rotas acima, este serviço também expõe endpoints compatíveis com o contrato do `IBillingServiceRepository`:

- `POST /v1/estimates` → cria orçamento (CreateEstimate)
- `PATCH /v1/estimates/approve` → aprova orçamento (ApproveEstimate)
- `PATCH /v1/estimates/reject` → rejeita orçamento (RejectEstimate)
- `PATCH /v1/estimates/cancel` → cancela orçamento (CancelEstimate)
- `GET /v1/payments/:estimate_id` → busca pagamento por orçamento (GetPaymentByEstimateID)
- `POST /v1/payments/:estimate_id` → cria pagamento (CreatePayment)

### Payload de estimate compatível

Para os endpoints de estimate compatíveis, o serviço aceita o payload `EstimateRequest` da integração e faz extração tolerante de dados:

- identificador da OS: `os_id`, `service_order_id`, ou `service_order.id`
- valor: `value_cents`, `valor`, `value`, ou somatório de `service_order` + `additional_repair` quando presentes

Isso permite manter compatibilidade sem acoplar a camada de domínio ao formato externo.

## Como rodar localmente

### Docker Compose (recomendado)

Subir API + DynamoDB Local + init das tabelas:

Use o profile `dynamodb-local`:

- `docker compose --profile dynamodb-local up --build`

API: <http://localhost:8080>

Swagger: <http://localhost:8080/swagger/index.html>

DynamoDB Local: `http://localhost:8000`

DynamoDB Admin (UI): <http://localhost:8001>

### (Opcional) LocalStack

Se preferir usar o endpoint estilo AWS (porta 4566), suba com profile:

```bash
docker compose --profile localstack up --build
```

> A API no profile `localstack` já inicia com `DYNAMODB_ENDPOINT=http://localstack:4566`.

### NoSQL Workbench

Crie uma conexão DynamoDB local apontando para:

- Endpoint: `http://localhost:8000`
- Region: `us-east-1`
- Access Key / Secret: qualquer valor (ex.: `local` / `local`)
