package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mecanica_xpto/internal/adapter/http/handlers/mocks"
	"mecanica_xpto/internal/domain/entities"
	"mecanica_xpto/internal/usecase"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

type failingReadCloser struct{}

func (failingReadCloser) Read(_ []byte) (int, error) { return 0, errors.New("read error") }
func (failingReadCloser) Close() error               { return nil }

func TestBillingPaymentHandler_CreatePaymentByEstimateID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PAYMENT_GATEWAY_MOCK", "")
	t.Setenv("MERCADOPAGO_MOCK", "")

	t.Run("invalid payload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIBillingPaymentUseCase(ctrl)
		h := NewBillingPaymentHandler(uc)

		r := gin.New()
		r.POST("/v1/payments/:estimate_id", h.CreatePaymentByEstimateID)

		req := httptest.NewRequest(http.MethodPost, "/v1/payments/est-1", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("usecase mapped error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIBillingPaymentUseCase(ctrl)
		h := NewBillingPaymentHandler(uc)

		r := gin.New()
		r.POST("/v1/payments/:estimate_id", h.CreatePaymentByEstimateID)

		uc.EXPECT().CreateAndApprove(gomock.Any(), "est-1", gomock.Any()).Return(entities.BillingPayment{}, usecase.ErrEstimateNotApproved)

		req := httptest.NewRequest(http.MethodPost, "/v1/payments/est-1", bytes.NewBufferString(`{"payment_method_id":"pix","payer":{"email":"x@test.com"}}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIBillingPaymentUseCase(ctrl)
		h := NewBillingPaymentHandler(uc)

		r := gin.New()
		r.POST("/v1/payments/:estimate_id", h.CreatePaymentByEstimateID)

		now := time.Now().UTC()
		uc.EXPECT().CreateAndApprove(gomock.Any(), "est-1", gomock.Any()).Return(entities.BillingPayment{ID: "pay-1", EstimateID: "est-1", Date: now, Status: entities.PaymentStatusAprovado}, nil)

		req := httptest.NewRequest(http.MethodPost, "/v1/payments/est-1", bytes.NewBufferString(`{"mp_payload":{"payment_method_id":"pix","payer":{"email":"x@test.com"}}}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body["payment_id"] != "pay-1" {
			t.Fatalf("unexpected body: %s", w.Body.String())
		}
	})
}

func TestBillingPaymentHandler_GetPaymentByEstimateID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PAYMENT_GATEWAY_MOCK", "")
	t.Setenv("MERCADOPAGO_MOCK", "")

	t.Run("list error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIBillingPaymentUseCase(ctrl)
		h := NewBillingPaymentHandler(uc)

		r := gin.New()
		r.GET("/v1/payments/:estimate_id", h.GetPaymentByEstimateID)

		uc.EXPECT().ListByEstimateID(gomock.Any(), "est-1").Return(nil, usecase.ErrInvalidPaymentEstimateID)

		req := httptest.NewRequest(http.MethodGet, "/v1/payments/est-1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIBillingPaymentUseCase(ctrl)
		h := NewBillingPaymentHandler(uc)

		r := gin.New()
		r.GET("/v1/payments/:estimate_id", h.GetPaymentByEstimateID)

		uc.EXPECT().ListByEstimateID(gomock.Any(), "est-1").Return([]entities.BillingPayment{}, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/payments/est-1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("success returns latest", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIBillingPaymentUseCase(ctrl)
		h := NewBillingPaymentHandler(uc)

		r := gin.New()
		r.GET("/v1/payments/:estimate_id", h.GetPaymentByEstimateID)

		old := entities.BillingPayment{ID: "old", EstimateID: "est-1", Date: time.Now().Add(-time.Hour), Status: entities.PaymentStatusPendente}
		latest := entities.BillingPayment{ID: "latest", EstimateID: "est-1", Date: time.Now(), Status: entities.PaymentStatusAprovado}
		uc.EXPECT().ListByEstimateID(gomock.Any(), "est-1").Return([]entities.BillingPayment{old, latest}, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/payments/est-1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body["payment_id"] != "latest" {
			t.Fatalf("expected latest payment, got body: %s", w.Body.String())
		}
	})
}

func TestReadMPPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PAYMENT_GATEWAY_MOCK", "")
	t.Setenv("MERCADOPAGO_MOCK", "")

	makeCtx := func(raw string) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(raw))
		c.Request.Header.Set("Content-Type", "application/json")
		return c
	}

	ctxReadErr := makeCtx("{}")
	ctxReadErr.Request.Body = failingReadCloser{}
	if _, err := readMPPayload(ctxReadErr); err == nil {
		t.Fatalf("expected read body error")
	}

	if _, err := readMPPayload(makeCtx("{invalid")); err == nil {
		t.Fatalf("expected invalid json error")
	}

	payload, err := readMPPayload(makeCtx("   "))
	if err != nil || string(payload) != "{}" {
		t.Fatalf("expected {}, got payload=%s err=%v", string(payload), err)
	}

	if _, err := readMPPayload(makeCtx(`{"mp_payload":null}`)); err == nil {
		t.Fatalf("expected mp_payload empty error")
	}

	payload, err = readMPPayload(makeCtx(`{"mp_payload":"x"}`))
	if err != nil || string(payload) != `"x"` {
		t.Fatalf("expected wrapped string payload, got %s err=%v", payload, err)
	}

	payload, err = readMPPayload(makeCtx(`{"mp_payload":{"a":1}}`))
	if err != nil || string(payload) != `{"a":1}` {
		t.Fatalf("expected wrapped payload, got %s err=%v", payload, err)
	}

	payload, err = readMPPayload(makeCtx(`{"mp_payload":{"payment_method_id":"pix"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(payload) {
		t.Fatalf("expected valid payload")
	}

	payload, err = readMPPayload(makeCtx(`{"payment_method_id":"pix"}`))
	if err != nil || string(payload) != `{"payment_method_id":"pix"}` {
		t.Fatalf("expected raw body payload, got %s err=%v", payload, err)
	}
}

func TestMapBillingPaymentError(t *testing.T) {
	t.Setenv("PAYMENT_GATEWAY_MOCK", "")
	t.Setenv("MERCADOPAGO_MOCK", "")

	cases := []struct {
		err  error
		code int
	}{
		{usecase.ErrInvalidPaymentEstimateID, http.StatusBadRequest},
		{usecase.ErrInvalidMPPayload, http.StatusBadRequest},
		{usecase.ErrPaymentGatewayBadRequest, http.StatusBadRequest},
		{usecase.ErrPaymentGatewayCustomerNotFound, http.StatusBadRequest},
		{usecase.ErrPaymentGatewayInvalidUsers, http.StatusBadRequest},
		{usecase.ErrPaymentGatewayUnauthorized, http.StatusUnauthorized},
		{usecase.ErrEstimateNotFound, http.StatusNotFound},
		{usecase.ErrEstimateNotApproved, http.StatusConflict},
		{usecase.ErrBillingPaymentNotFound, http.StatusNotFound},
		{errors.New("other"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		got := mapBillingPaymentError(tc.err)
		if got.HTTPStatus != tc.code {
			t.Fatalf("for err %v expected %d got %d", tc.err, tc.code, got.HTTPStatus)
		}
	}
}
