# Testes locais com Insomnia (Billing Service + DynamoDB + Mercado Pago)

Este documento descreve como testar **todas as rotas** do Billing Service usando o **Insomnia**, incluindo uma **simulação completa** do fluxo de pagamento via **Mercado Pago**.

> Base path da API: `/v1`

---

## 1) Pré-requisitos

### 1.1 Subir o ambiente via Docker

O projeto suporta 2 modos:

- **DynamoDB Local** (recomendado para persistência local simples)
- **LocalStack** (recomendado quando você quer simular o endpoint AWS `:4566`)

**DynamoDB Local (profile `dynamodb-local`)**
- API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`
- DynamoDB Local: `http://localhost:8000`
- DynamoDB Admin UI: `http://localhost:8001`

**LocalStack (profile `localstack`)**
- API: `http://localhost:8080`
- LocalStack: `http://localhost:4566`

> Observação: o script de init cria as tabelas e também **faz seed de 1 registro em cada tabela**.

### 1.2 Variáveis de ambiente relevantes

No arquivo [.env-example](.env-example) você encontra o modelo. As principais variáveis são:

- `DYNAMODB_ENDPOINT`
  - DynamoDB Local: `http://dynamodb:8000` (dentro do compose)
  - LocalStack: `http://localstack:4566` (dentro do compose)
- `ESTIMATES_TABLE` (default: `estimates`)
- `PAYMENTS_TABLE` (default: `payments`)

Para Mercado Pago:
- `MERCADOPAGO_ACCESS_TOKEN`
  - **Obrigatório** para o `POST /v1/payments` realmente chamar a API do Mercado Pago.

Seeds (opcional):
- `SEED_OS_ID` (default: `os_demo_1`)
- `SEED_ESTIMATE_VALUE_CENTS` (default: `10000`)
- `SEED_PAYMENT_ID` (default: `pay_demo_1`)

---

## 2) Configurando o Insomnia

### 2.1 Crie um Environment (Workspace)

Sugestão de Environment JSON:

- `base_url`: `http://localhost:8080/v1`
- `os_id`: `os_demo_1`
- `estimate_id`: `os_demo_1` *(neste serviço, o `estimate_id` é o próprio `os_id`)*

Exemplo:

```json
{
  "base_url": "http://localhost:8080/v1",
  "os_id": "os_demo_1",
  "estimate_id": "os_demo_1"
}
```

### 2.2 Headers

As rotas atuais **não exigem autenticação**.

Use apenas:
- `Content-Type: application/json`

---

## 3) Rotas e comportamento esperado

## 3.1 Health check

### Request
- **GET** `{{ base_url }}/ping`

### Response (200)
```json
{ "message": "pong" }
```

---

## 3.2 Criar orçamento (Estimate) — “Calcula Orçamento”

### Request
- **POST** `{{ base_url }}/os/{{ os_id }}/estimate`

Body (envie **um** dos campos):

**Opção A (preferida)**
```json
{ "value_cents": 12500 }
```

**Opção B (decimal em BRL, será convertido para cents)**
```json
{ "valor": 125.00 }
```

### Regras
- Cria o estimate com:
  - `id = os_id`
  - `status = "pendente"`
- Só permite **1 estimate por OS**.

### Response (201)
```json
{
  "id": "os_demo_1",
  "os_id": "os_demo_1",
  "value_cents": 12500,
  "valor": 125,
  "status": "pendente",
  "created_at": "2026-02-12T02:10:00Z",
  "updated_at": "2026-02-12T02:10:00Z"
}
```

### Erros comuns
- **400** `INVALID_REQUEST` (payload inválido / valor <= 0)
- **409** `ESTIMATE_ALREADY_EXISTS` (já existe estimate para esse `os_id`)

### Payload no DynamoDB (tabela `estimates`)
Representação esperada (campos principais):
```json
{
  "id": "os_demo_1",
  "os_id": "os_demo_1",
  "value_cents": 12500,
  "status": "pendente",
  "created_at": "2026-02-12T02:10:00Z",
  "updated_at": "2026-02-12T02:10:00Z"
}
```

---

## 3.3 Atualizar status do orçamento (aprovar / rejeitar / cancelar)

### Request
- **PATCH** `{{ base_url }}/os/{{ os_id }}/estimate`

Body:
```json
{ "acao": "aprovar" }
```

Ações aceitas:
- `aprovar`
- `rejeitar`
- `cancelar`

### Response (200)
```json
{
  "id": "os_demo_1",
  "os_id": "os_demo_1",
  "value_cents": 12500,
  "valor": 125,
  "status": "aprovado",
  "created_at": "2026-02-12T02:10:00Z",
  "updated_at": "2026-02-12T02:12:00Z"
}
```

### Erros comuns
- **400** `INVALID_ACTION` (ação diferente de aprovar/rejeitar/cancelar)
- **404** `ESTIMATE_NOT_FOUND` (não existe estimate para esse `os_id`)

### Payload no DynamoDB (mudança esperada)
```json
{
  "id": "os_demo_1",
  "status": "aprovado",
  "updated_at": "2026-02-12T02:12:00Z"
}
```

---

## 3.4 Recalcular orçamento total (atualizar valor)

### Request
- **PATCH** `{{ base_url }}/estimates/{{ estimate_id }}`

Body (envie **um** dos campos):

```json
{ "value_cents": 15000 }
```

ou

```json
{ "valor": 150.00 }
```

### Response (200)
```json
{
  "id": "os_demo_1",
  "os_id": "os_demo_1",
  "value_cents": 15000,
  "valor": 150,
  "status": "aprovado",
  "created_at": "2026-02-12T02:10:00Z",
  "updated_at": "2026-02-12T02:15:00Z"
}
```

### Erros comuns
- **404** `ESTIMATE_NOT_FOUND`
- **400** `INVALID_REQUEST` (valor inválido)

### Payload no DynamoDB (mudança esperada)
```json
{
  "id": "os_demo_1",
  "value_cents": 15000,
  "updated_at": "2026-02-12T02:15:00Z"
}
```

---

# 4) Pagamento (Mercado Pago) — Simulação completa

## 4.1 Visão geral do que acontece

Quando você chama **`POST /v1/payments`**:

1) A API valida `estimate_id` e `mp_payload`.
2) Busca o estimate no DynamoDB.
3) **Exige** que o estimate exista e esteja com `status = "aprovado"`.
4) Envia o `mp_payload` para o endpoint do Mercado Pago via SDK oficial Go (`mercadopago/sdk-go`).
5) Recebe a resposta do Mercado Pago (inclui `id` e `status`).
6) Persiste em `payments`:
   - `id` = ID retornado pelo Mercado Pago (convertido para string)
   - `estimate_id` = seu estimate
   - `status` mapeado para o domínio:
     - MP `approved` -> `aprovado`
     - MP `rejected`/`cancelled`/... -> `negado`
     - outros -> `pendente`
   - `mp_payload_raw` = JSON completo da resposta do MP
   - `mp_payload` = versão parseada (map) da resposta (best-effort)

> Importante: este serviço **ainda não implementa webhook** para atualizar pagamentos assíncronos. O estado persistido é o que o MP devolveu no momento do `Create`.

---

## 4.2 Pré-condições para testar pagamento

Antes de chamar `POST /v1/payments`:

1) Crie o estimate: `POST /os/{os_id}/estimate`
2) Aprove o estimate: `PATCH /os/{os_id}/estimate` com `{ "acao": "aprovar" }`
3) Configure `MERCADOPAGO_ACCESS_TOKEN` no `.env` e suba os containers.

---

## 4.3 Rota de pagamento

### Request
- **POST** `{{ base_url }}/payments/`

Body:
```json
{
  "estimate_id": "{{ estimate_id }}",
  "mp_payload": {
    "description": "Pagamento do estimate os_demo_1",
    "payment_method_id": "pix",
    "transaction_amount": 150.00,
    "payer": {
      "email": "comprador_teste@exemplo.com"
    }
  }
}
```

#### Campos que a API completa automaticamente (se ausentes)
Se você **não** enviar estes campos dentro de `mp_payload`, a API tenta preencher:

- `external_reference`: `estimate_id`
- `description`: `"Estimate <estimate_id>"`
- `transaction_amount`: `estimate.value_cents / 100`

> Dica: envie `transaction_amount` igual ao estimate, para evitar divergência.

---

### Response (201)
A resposta é o payload do seu serviço (não é o payload nativo do MP), mas inclui os dados do MP em `mp_payload_raw` e `mp_payload`.

Exemplo (campos variam conforme o meio de pagamento):

```json
{
  "id": "1234567890",
  "estimate_id": "os_demo_1",
  "date": "2026-02-12T02:20:00Z",
  "status": "pendente",
  "mp_payload_raw": "{...json completo retornado pelo Mercado Pago...}",
  "mp_payload": {
    "id": 1234567890,
    "status": "pending",
    "status_detail": "pending_waiting_payment",
    "transaction_amount": 150,
    "external_reference": "os_demo_1"
  }
}
```

### Erros comuns
- **400** `INVALID_REQUEST`
  - `estimate_id` vazio
  - `mp_payload` vazio / não-JSON
- **404** `ESTIMATE_NOT_FOUND`
  - não existe estimate para esse id
- **409** `ESTIMATE_NOT_APPROVED`
  - estimate existe mas não está `aprovado`
- **500** `INTERNAL_ERROR`
  - problemas de credencial (`MERCADOPAGO_ACCESS_TOKEN` ausente)
  - erro HTTP do Mercado Pago
  - payload incompatível com o endpoint de pagamento

---

## 4.4 Payload no DynamoDB (tabela `payments`)

O repositório persiste com este “shape” (principais campos):

```json
{
  "id": "1234567890",
  "estimate_id": "os_demo_1",
  "date": "2026-02-12T02:20:00Z",
  "status": "pendente",
  "mp_payload_raw": "{...json do MP...}",
  "mp_payload": {
    "id": 1234567890,
    "status": "pending",
    "external_reference": "os_demo_1"
  }
}
```

> Observação: `mp_payload_raw` é **string** (JSON serializado). `mp_payload` é **map**.

---

## 4.5 Exemplos de `mp_payload` (para Insomnia)

### A) Pix (geralmente o mais simples para testar)
```json
{
  "payment_method_id": "pix",
  "transaction_amount": 150.00,
  "description": "Pagamento PIX do estimate os_demo_1",
  "external_reference": "os_demo_1",
  "payer": {
    "email": "comprador_teste@exemplo.com"
  }
}
```

**O que esperar**
- O Mercado Pago tende a retornar `status` como `pending`/`in_process`.
- O seu serviço deve persistir como `pendente`.

### B) Cartão (requer `token` do Mercado Pago)
```json
{
  "token": "{{CARD_TOKEN}}",
  "payment_method_id": "visa",
  "transaction_amount": 150.00,
  "installments": 1,
  "description": "Pagamento cartão do estimate os_demo_1",
  "external_reference": "os_demo_1",
  "payer": {
    "email": "comprador_teste@exemplo.com"
  }
}
```

**O que esperar**
- Se o token/cartão de teste estiver correto, o MP pode retornar `status` `approved`.
- O seu serviço deve persistir como `aprovado`.

---

## 4.6 Fluxo completo (checklist no Insomnia)

1) **Ping**
   - `GET /ping` -> 200
2) **Criar estimate**
   - `POST /os/{{os_id}}/estimate` -> 201 (`status=pendente`)
3) **Aprovar estimate**
   - `PATCH /os/{{os_id}}/estimate` `{ "acao":"aprovar" }` -> 200 (`status=aprovado`)
4) **(Opcional) Recalcular valor**
   - `PATCH /estimates/{{estimate_id}}` -> 200
5) **Criar pagamento no MP via serviço**
   - `POST /payments` -> 201
6) **Validar persistência**
   - Veja as tabelas na DynamoDB Admin UI (quando em DynamoDB Local) ou via ferramentas do LocalStack.

---

## 5) Observações importantes

- O `POST /v1/payments` envia o `mp_payload` para o Mercado Pago como `payment.Request`.
  - Se o MP retornar erro de validação (400), ajuste o `mp_payload` para incluir os campos exigidos para o meio de pagamento escolhido.
- O serviço persiste o JSON completo da resposta do MP em `mp_payload_raw`, útil para auditoria.
- Se você rodar com **LocalStack**, use `DYNAMODB_ENDPOINT` apontando para `http://localstack:4566` (isso já é default do serviço `app-localstack`).
