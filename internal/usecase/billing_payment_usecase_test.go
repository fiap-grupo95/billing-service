package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"mecanica_xpto/internal/domain/entities"
	mock_interfaces "mecanica_xpto/internal/usecase/interfaces/mocks"

	"go.uber.org/mock/gomock"
)

func TestBillingPaymentUseCase_CreateAndApprove_Validations(t *testing.T) {
	t.Run("empty estimate id", func(t *testing.T) {
		uc := NewBillingPaymentUseCase(nil, nil, nil)
		_, err := uc.CreateAndApprove(context.Background(), " ", json.RawMessage(`{}`))
		if !errors.Is(err, ErrInvalidPaymentEstimateID) {
			t.Fatalf("expected ErrInvalidPaymentEstimateID, got %v", err)
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		uc := NewBillingPaymentUseCase(nil, nil, nil)
		_, err := uc.CreateAndApprove(context.Background(), "est-1", nil)
		if !errors.Is(err, ErrInvalidMPPayload) {
			t.Fatalf("expected ErrInvalidMPPayload, got %v", err)
		}
	})

	t.Run("invalid json payload", func(t *testing.T) {
		uc := NewBillingPaymentUseCase(nil, nil, nil)
		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{`))
		if !errors.Is(err, ErrInvalidMPPayload) {
			t.Fatalf("expected ErrInvalidMPPayload, got %v", err)
		}
	})

	t.Run("gateway not configured", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		uc := NewBillingPaymentUseCase(nil, estRepo, nil)

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix"}`))
		if err == nil || err.Error() != "payment gateway not configured" {
			t.Fatalf("expected gateway not configured error, got %v", err)
		}
	})

	t.Run("estimate repository not configured", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(nil, nil, gateway)

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix"}`))
		if err == nil || err.Error() != "estimate repository not configured" {
			t.Fatalf("expected estimate repository not configured error, got %v", err)
		}
	})
}

func TestBillingPaymentUseCase_CreateAndApprove_EstimateChecks(t *testing.T) {
	t.Run("estimate repo returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(repo, estRepo, gateway)

		estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{}, errors.New("db"))

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix"}`))
		if err == nil || err.Error() != "db" {
			t.Fatalf("expected db error, got %v", err)
		}
	})

	t.Run("estimate not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(repo, estRepo, gateway)

		estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{}, nil)

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix"}`))
		if !errors.Is(err, ErrEstimateNotFound) {
			t.Fatalf("expected ErrEstimateNotFound, got %v", err)
		}
	})

	t.Run("estimate not approved", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(repo, estRepo, gateway)

		estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{ID: "est-1", Status: entities.EstimateStatusPendente}, nil)

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix"}`))
		if !errors.Is(err, ErrEstimateNotApproved) {
			t.Fatalf("expected ErrEstimateNotApproved, got %v", err)
		}
	})
}

func TestBillingPaymentUseCase_CreateAndApprove_PayloadValidation(t *testing.T) {
	t.Run("missing payment_method_id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(repo, estRepo, gateway)

		estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{ID: "est-1", Status: entities.EstimateStatusAprovado}, nil)

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payer":{"email":"x@test.com"}}`))
		if !errors.Is(err, ErrInvalidMPPayload) {
			t.Fatalf("expected ErrInvalidMPPayload, got %v", err)
		}
	})

	t.Run("missing payer", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(repo, estRepo, gateway)
		t.Setenv("MERCADOPAGO_ACCESS_TOKEN", "")
		t.Setenv("MERCADOPAGO_TEST_PAYER_EMAIL", "")

		estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{ID: "est-1", Status: entities.EstimateStatusAprovado}, nil)

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix"}`))
		if !errors.Is(err, ErrInvalidMPPayload) {
			t.Fatalf("expected ErrInvalidMPPayload, got %v", err)
		}
	})
}

func TestBillingPaymentUseCase_CreateAndApprove_GatewayErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{name: "customer not found", err: errors.New(`{"code":2002}`), want: ErrPaymentGatewayCustomerNotFound},
		{name: "invalid users", err: errors.New(`invalid users involved`), want: ErrPaymentGatewayInvalidUsers},
		{name: "unauthorized", err: errors.New(`{"error":"unauthorized"}`), want: ErrPaymentGatewayUnauthorized},
		{name: "bad request", err: errors.New(`{"status":400}`), want: ErrPaymentGatewayBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
			estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
			uc := NewBillingPaymentUseCase(repo, estRepo, gateway)

			estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{ID: "est-1", Status: entities.EstimateStatusAprovado, Price: 10}, nil)
			gateway.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return("", "", nil, tc.err)

			_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix","payer":{"email":"x@test.com"}}`))
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}

	t.Run("unknown gateway error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(repo, estRepo, gateway)

		estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{ID: "est-1", Status: entities.EstimateStatusAprovado, Price: 10}, nil)
		gateway.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return("", "", nil, errors.New("boom"))

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix","payer":{"email":"x@test.com"}}`))
		if err == nil || err.Error() != "boom" {
			t.Fatalf("expected boom, got %v", err)
		}
	})
}

func TestBillingPaymentUseCase_CreateAndApprove_SuccessAndStatuses(t *testing.T) {
	cases := []struct {
		name           string
		providerStatus string
		want           entities.PaymentStatus
		providerResp   json.RawMessage
	}{
		{name: "approved", providerStatus: "approved", want: entities.PaymentStatusAprovado, providerResp: json.RawMessage(`{"id":123}`)},
		{name: "rejected", providerStatus: "rejected", want: entities.PaymentStatusNegado, providerResp: json.RawMessage(`{"id":123}`)},
		{name: "pending default", providerStatus: "in_process", want: entities.PaymentStatusPendente, providerResp: json.RawMessage(`{"id":123}`)},
		{name: "invalid provider response json", providerStatus: "approved", want: entities.PaymentStatusAprovado, providerResp: json.RawMessage(`{`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
			estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
			gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
			uc := NewBillingPaymentUseCase(repo, estRepo, gateway)
			t.Setenv("MERCADOPAGO_ACCESS_TOKEN", "TEST-token")
			t.Setenv("MERCADOPAGO_TEST_PAYER_USER_ID", "123")
			t.Setenv("MERCADOPAGO_TEST_PAYER_EMAIL", "sandbox@test.com")

			estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{ID: "est-1", Status: entities.EstimateStatusAprovado, Price: 77.2}, nil)

			gateway.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, payload json.RawMessage) (string, string, json.RawMessage, error) {
					var body map[string]any
					if err := json.Unmarshal(payload, &body); err != nil {
						t.Fatalf("payload should be valid json: %v", err)
					}
					if body["external_reference"] != "est-1" {
						t.Fatalf("external_reference not set")
					}
					if body["description"] != "Estimate est-1" {
						t.Fatalf("description not set")
					}
					if body["transaction_amount"] != float64(77.2) {
						t.Fatalf("transaction_amount should come from estimate")
					}
					payer := body["payer"].(map[string]any)
					if payer["email"] == nil {
						t.Fatalf("expected payer email fallback/mapping")
					}
					return "pay-1", tc.providerStatus, tc.providerResp, nil
				},
			)

			repo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(entities.BillingPayment{})).DoAndReturn(
				func(_ context.Context, p entities.BillingPayment) (entities.BillingPayment, error) {
					if p.ID != "pay-1" || p.EstimateID != "est-1" || p.Status != tc.want {
						t.Fatalf("unexpected payment: %+v", p)
					}
					if p.Date.IsZero() {
						t.Fatalf("date must be set")
					}
					return p, nil
				},
			)

			res, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix","payer":{"id":"123"}}`))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Status != tc.want {
				t.Fatalf("expected status %s, got %s", tc.want, res.Status)
			}
		})
	}

	t.Run("repository create error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(repo, estRepo, gateway)

		estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{ID: "est-1", Status: entities.EstimateStatusAprovado, Price: 11}, nil)
		gateway.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return("pay-1", "approved", json.RawMessage(`{"id":123}`), nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.BillingPayment{}, errors.New("db-create"))

		_, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`{"payment_method_id":"pix","payer":{"email":"x@test.com"}}`))
		if err == nil || err.Error() != "db-create" {
			t.Fatalf("expected db-create error, got %v", err)
		}
	})
}

func TestBillingPaymentUseCase_Getters(t *testing.T) {
	t.Run("GetByID invalid", func(t *testing.T) {
		uc := NewBillingPaymentUseCase(nil, nil, nil)
		_, err := uc.GetByID(context.Background(), "")
		if err == nil || err.Error() != "invalid payment id" {
			t.Fatalf("expected invalid payment id, got %v", err)
		}
	})

	t.Run("GetByID repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		uc := NewBillingPaymentUseCase(repo, nil, nil)
		repo.EXPECT().GetByID(gomock.Any(), "id-1").Return(entities.BillingPayment{}, errors.New("db"))

		_, err := uc.GetByID(context.Background(), "id-1")
		if err == nil || err.Error() != "db" {
			t.Fatalf("expected db error, got %v", err)
		}
	})

	t.Run("GetByID not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		uc := NewBillingPaymentUseCase(repo, nil, nil)
		repo.EXPECT().GetByID(gomock.Any(), "id-1").Return(entities.BillingPayment{}, nil)

		_, err := uc.GetByID(context.Background(), "id-1")
		if !errors.Is(err, ErrBillingPaymentNotFound) {
			t.Fatalf("expected ErrBillingPaymentNotFound, got %v", err)
		}
	})

	t.Run("GetByID success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		uc := NewBillingPaymentUseCase(repo, nil, nil)
		repo.EXPECT().GetByID(gomock.Any(), "id-1").Return(entities.BillingPayment{ID: "id-1"}, nil)

		res, err := uc.GetByID(context.Background(), " id-1 ")
		if err != nil || res.ID != "id-1" {
			t.Fatalf("unexpected result err=%v res=%+v", err, res)
		}
	})

	t.Run("ListByEstimateID invalid", func(t *testing.T) {
		uc := NewBillingPaymentUseCase(nil, nil, nil)
		_, err := uc.ListByEstimateID(context.Background(), " ")
		if !errors.Is(err, ErrInvalidPaymentEstimateID) {
			t.Fatalf("expected ErrInvalidPaymentEstimateID, got %v", err)
		}
	})

	t.Run("ListByEstimateID success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		uc := NewBillingPaymentUseCase(repo, nil, nil)
		expected := []entities.BillingPayment{{ID: "p1", Date: time.Now()}}
		repo.EXPECT().ListByEstimateID(gomock.Any(), "est-1").Return(expected, nil)

		res, err := uc.ListByEstimateID(context.Background(), " est-1 ")
		if err != nil || len(res) != 1 || res[0].ID != "p1" {
			t.Fatalf("unexpected result err=%v res=%+v", err, res)
		}
	})
}

func TestBillingPaymentUseCase_HelperFunctions(t *testing.T) {
	t.Run("hasNonEmptyString", func(t *testing.T) {
		if hasNonEmptyString(map[string]any{}, "x") {
			t.Fatalf("expected false")
		}
		if hasNonEmptyString(map[string]any{"x": 1}, "x") {
			t.Fatalf("expected false for non-string")
		}
		if hasNonEmptyString(map[string]any{"x": "   "}, "x") {
			t.Fatalf("expected false for empty string")
		}
		if !hasNonEmptyString(map[string]any{"x": "ok"}, "x") {
			t.Fatalf("expected true")
		}
	})

	t.Run("hasPayer and hasPayerID", func(t *testing.T) {
		if hasPayer(map[string]any{}) {
			t.Fatalf("expected false")
		}
		if hasPayer(map[string]any{"payer": "x"}) {
			t.Fatalf("expected false")
		}
		if hasPayer(map[string]any{"payer": map[string]any{}}) {
			t.Fatalf("expected false")
		}
		if !hasPayer(map[string]any{"payer": map[string]any{"email": "a@b.com"}}) {
			t.Fatalf("expected true with email")
		}
		if !hasPayer(map[string]any{"payer": map[string]any{"id": 10}}) {
			t.Fatalf("expected true with id")
		}
		if hasPayerID(map[string]any{"id": nil}) {
			t.Fatalf("expected false for nil id")
		}
		if hasPayerID(map[string]any{"id": " "}) {
			t.Fatalf("expected false for blank id")
		}
	})

	t.Run("ensurePayerDefaults", func(t *testing.T) {
		t.Setenv("MERCADOPAGO_TEST_PAYER_EMAIL", "")
		t.Setenv("MERCADOPAGO_ACCESS_TOKEN", "")
		m := map[string]any{}
		ensurePayerDefaults(m)
		payer := m["payer"].(map[string]any)
		if payer["type"] != "customer" {
			t.Fatalf("expected type customer")
		}

		m2 := map[string]any{"payer": map[string]any{}}
		t.Setenv("MERCADOPAGO_TEST_PAYER_EMAIL", "custom@test.com")
		ensurePayerDefaults(m2)
		payer2 := m2["payer"].(map[string]any)
		if payer2["email"] != "custom@test.com" {
			t.Fatalf("expected env email fallback")
		}

		m3 := map[string]any{"payer": map[string]any{}}
		t.Setenv("MERCADOPAGO_TEST_PAYER_EMAIL", "")
		t.Setenv("MERCADOPAGO_ACCESS_TOKEN", "TEST-123")
		ensurePayerDefaults(m3)
		payer3 := m3["payer"].(map[string]any)
		if payer3["email"] != "test_user_br@testuser.com" {
			t.Fatalf("expected sandbox fallback email")
		}

		m4 := map[string]any{"payer": "invalid"}
		ensurePayerDefaults(m4)
	})

	t.Run("normalizeSandboxPayerFromUserID", func(t *testing.T) {
		m := map[string]any{}
		normalizeSandboxPayerFromUserID(m)

		m2 := map[string]any{"payer": "invalid"}
		normalizeSandboxPayerFromUserID(m2)

		t.Setenv("MERCADOPAGO_ACCESS_TOKEN", "APP-123")
		m3 := map[string]any{"payer": map[string]any{"id": "123"}}
		normalizeSandboxPayerFromUserID(m3)
		if _, ok := m3["payer"].(map[string]any)["email"]; ok {
			t.Fatalf("should not map for non TEST token")
		}

		t.Setenv("MERCADOPAGO_ACCESS_TOKEN", "TEST-123")
		t.Setenv("MERCADOPAGO_TEST_PAYER_USER_ID", "")
		t.Setenv("MERCADOPAGO_TEST_PAYER_EMAIL", "")
		mCfgMissing := map[string]any{"payer": map[string]any{"id": "123"}}
		normalizeSandboxPayerFromUserID(mCfgMissing)
		if _, ok := mCfgMissing["payer"].(map[string]any)["email"]; ok {
			t.Fatalf("should not map when env config is missing")
		}

		t.Setenv("MERCADOPAGO_TEST_PAYER_USER_ID", "123")
		t.Setenv("MERCADOPAGO_TEST_PAYER_EMAIL", "sandbox@test.com")
		m4 := map[string]any{"payer": map[string]any{"id": "999"}}
		normalizeSandboxPayerFromUserID(m4)
		if _, ok := m4["payer"].(map[string]any)["email"]; ok {
			t.Fatalf("should not map mismatched id")
		}

		m5 := map[string]any{"payer": map[string]any{"id": "123"}}
		normalizeSandboxPayerFromUserID(m5)
		payer := m5["payer"].(map[string]any)
		if payer["email"] != "sandbox@test.com" {
			t.Fatalf("expected mapped email")
		}
		if _, ok := payer["id"]; ok {
			t.Fatalf("expected id removed")
		}
	})

	t.Run("create and approve with non-object payload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := mock_interfaces.NewMockIBillingPaymentRepository(ctrl)
		estRepo := mock_interfaces.NewMockIEstimateRepository(ctrl)
		gateway := mock_interfaces.NewMockIPaymentGateway(ctrl)
		uc := NewBillingPaymentUseCase(repo, estRepo, gateway)

		estRepo.EXPECT().GetByID(gomock.Any(), "est-1").Return(entities.Estimate{ID: "est-1", Status: entities.EstimateStatusAprovado, Price: 42}, nil)
		gateway.EXPECT().CreatePayment(gomock.Any(), json.RawMessage(`[]`)).Return("pay-1", "approved", json.RawMessage(`{"id":1}`), nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.BillingPayment{ID: "pay-1", EstimateID: "est-1", Status: entities.PaymentStatusAprovado}, nil)

		res, err := uc.CreateAndApprove(context.Background(), "est-1", json.RawMessage(`[]`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ID != "pay-1" {
			t.Fatalf("unexpected result: %+v", res)
		}
	})

	t.Run("gateway helper classifiers", func(t *testing.T) {
		if isGatewayBadRequest(nil) || isGatewayUnauthorized(nil) || isGatewayInvalidUsers(nil) || isGatewayCustomerNotFound(nil) {
			t.Fatalf("all nil checks should be false")
		}
		if !isGatewayBadRequest(errors.New(`{"error":"bad_request"}`)) {
			t.Fatalf("expected bad request true")
		}
		if !isGatewayUnauthorized(errors.New(`{"status":401}`)) {
			t.Fatalf("expected unauthorized true")
		}
		if !isGatewayInvalidUsers(errors.New(`{"code":2034}`)) {
			t.Fatalf("expected invalid users true")
		}
		if !isGatewayCustomerNotFound(errors.New(`customer not found`)) {
			t.Fatalf("expected customer not found true")
		}
	})
}
