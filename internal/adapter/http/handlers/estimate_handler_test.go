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

func TestEstimateHandler_CreateEstimate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("invalid json", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)

		r := gin.New()
		r.POST("/v1/estimates", h.CreateEstimate)

		req := httptest.NewRequest(http.MethodPost, "/v1/estimates", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing os id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)

		r := gin.New()
		r.POST("/v1/estimates", h.CreateEstimate)

		req := httptest.NewRequest(http.MethodPost, "/v1/estimates", bytes.NewBufferString(`{"service_order_id":"   ","services":[{"price":10}]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid price", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)

		r := gin.New()
		r.POST("/v1/estimates", h.CreateEstimate)

		req := httptest.NewRequest(http.MethodPost, "/v1/estimates", bytes.NewBufferString(`{"service_order_id":"os-1"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("usecase returns mapped error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)

		r := gin.New()
		r.POST("/v1/estimates", h.CreateEstimate)

		uc.EXPECT().CalculateEstimate(gomock.Any(), "os-1", 10.0).Return(entities.Estimate{}, usecase.ErrEstimateAlreadyExists)

		req := httptest.NewRequest(http.MethodPost, "/v1/estimates", bytes.NewBufferString(`{"service_order_id":"os-1","services":[{"price":10}]}`))
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
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)

		r := gin.New()
		r.POST("/v1/estimates", h.CreateEstimate)

		now := time.Now().UTC()
		uc.EXPECT().CalculateEstimate(gomock.Any(), "os-1", 10.0).Return(entities.Estimate{ID: "est-1", OSID: "os-1", Price: 10, Status: entities.EstimateStatusPendente, CreatedAt: now, UpdatedAt: now}, nil)

		req := httptest.NewRequest(http.MethodPost, "/v1/estimates", bytes.NewBufferString(`{"service_order_id":"os-1","services":[{"price":10}]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body["estimate_id"] != "est-1" {
			t.Fatalf("unexpected response body: %s", w.Body.String())
		}
	})
}

func TestEstimateHandler_PatchStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	build := func(method, path string, f gin.HandlerFunc) (*gin.Engine, string) {
		r := gin.New()
		r.PATCH(path, f)
		return r, path
	}

	t.Run("approve success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)
		r, path := build(http.MethodPatch, "/v1/estimates/approve", h.ApproveEstimate)

		uc.EXPECT().ApproveByOSID(gomock.Any(), "os-1").Return(entities.Estimate{ID: "est-1", OSID: "os-1", Status: entities.EstimateStatusAprovado}, nil)

		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(`{"service_order_id":"os-1","services":[{"price":1}]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("reject invalid json", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)
		r, path := build(http.MethodPatch, "/v1/estimates/reject", h.RejectEstimate)

		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("cancel missing os id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)
		r, path := build(http.MethodPatch, "/v1/estimates/cancel", h.CancelEstimate)

		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(`{"service_order_id":"  ","services":[{"price":1}]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("approve mapped error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockIEstimateUseCase(ctrl)
		h := NewEstimateHandler(uc)
		r, path := build(http.MethodPatch, "/v1/estimates/approve", h.ApproveEstimate)

		uc.EXPECT().ApproveByOSID(gomock.Any(), "os-1").Return(entities.Estimate{}, usecase.ErrEstimateNotFound)

		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(`{"service_order_id":"os-1","services":[{"price":1}]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestMapEstimateError(t *testing.T) {
	if got := mapEstimateError(usecase.ErrInvalidOSID); got.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
	if got := mapEstimateError(usecase.ErrInvalidEstimateID); got.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
	if got := mapEstimateError(usecase.ErrInvalidEstimateVal); got.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
	if got := mapEstimateError(usecase.ErrEstimateAlreadyExists); got.HTTPStatus != http.StatusConflict {
		t.Fatalf("expected 409")
	}
	if got := mapEstimateError(usecase.ErrEstimateNotFound); got.HTTPStatus != http.StatusNotFound {
		t.Fatalf("expected 404")
	}
	if got := mapEstimateError(errors.New("x")); got.HTTPStatus != http.StatusInternalServerError {
		t.Fatalf("expected 500")
	}
}
