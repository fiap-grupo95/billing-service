package pkg

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAppErrorError(t *testing.T) {
	internalErr := errors.New("internal problem")

	tests := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name: "Com erro interno",
			appErr: &AppError{
				Code:    "CODE1",
				Message: "msg1",
				Layer:   LayerDomain,
				Err:     internalErr,
			},
			expected: "Layer: domain, Code: CODE1, Message: msg1, Internal Error: internal problem",
		},
		{
			name: "Sem erro interno",
			appErr: &AppError{
				Code:    "CODE2",
				Message: "msg2",
				Layer:   LayerApplication,
				Err:     nil,
			},
			expected: "[application] CODE2: msg2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.appErr.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	internalErr := errors.New("internal")
	appErr := &AppError{
		Err: internalErr,
	}

	if !errors.Is(appErr.Unwrap(), internalErr) {
		t.Errorf("Unwrap() não retornou erro interno esperado")
	}
}

func TestAppErrorToHTTPError(t *testing.T) {
	appErr := &AppError{
		Code:    "CODE123",
		Message: "message here",
	}

	got := appErr.ToHTTPError()
	if got.Code != "CODE123" || got.Message != "message here" {
		t.Errorf("ToHTTPError() retornou %+v, esperado Code=CODE123, Message=message here", got)
	}
}

func TestAppErrorToJSON(t *testing.T) {
	appErr := &AppError{
		Code:    "C123",
		Message: "msg",
	}

	jsonBytes := appErr.ToJSON()
	var resp ErrorResponse
	if err := json.Unmarshal(jsonBytes, &resp); err != nil {
		t.Fatalf("Erro ao deserializar JSON: %v", err)
	}
	if resp.Code != "C123" || resp.Message != "msg" {
		t.Errorf("ToJSON() retornou %+v, esperado Code=C123, Message=msg", resp)
	}
}

func TestFactoryFunctions(t *testing.T) {
	internalErr := errors.New("internal error")

	tests := []struct {
		name       string
		factory    func() *AppError
		wantLayer  Layer
		wantCode   string
		wantMsg    string
		wantErr    error
		wantStatus int
	}{
		{
			name: "NewDomainError",
			factory: func() *AppError {
				return NewDomainError("D1", "domain error", internalErr, 400)
			},
			wantLayer:  LayerDomain,
			wantCode:   "D1",
			wantMsg:    "domain error",
			wantErr:    internalErr,
			wantStatus: 400,
		},
		{
			name: "NewDomainErrorSimple",
			factory: func() *AppError {
				return NewDomainErrorSimple("D2", "domain simple error", 404)
			},
			wantLayer:  LayerDomain,
			wantCode:   "D2",
			wantMsg:    "domain simple error",
			wantErr:    nil,
			wantStatus: 404,
		},
		{
			name: "NewApplicationError",
			factory: func() *AppError {
				return NewApplicationError("A1", "app error", internalErr, 401)
			},
			wantLayer:  LayerApplication,
			wantCode:   "A1",
			wantMsg:    "app error",
			wantErr:    internalErr,
			wantStatus: 401,
		},
		{
			name: "NewInfraError",
			factory: func() *AppError {
				return NewInfraError("I1", "infra error", internalErr, 500)
			},
			wantLayer:  LayerInfrastructure,
			wantCode:   "I1",
			wantMsg:    "infra error",
			wantErr:    internalErr,
			wantStatus: 500,
		},
	}

	checkAppError := func(t *testing.T, e *AppError, wantLayer Layer, wantCode, wantMsg string, wantErr error, wantStatus int) {
		if e.Layer != wantLayer {
			t.Errorf("Layer = %v, want %v", e.Layer, wantLayer)
		}
		if e.Code != wantCode {
			t.Errorf("Code = %v, want %v", e.Code, wantCode)
		}
		if e.Message != wantMsg {
			t.Errorf("Message = %v, want %v", e.Message, wantMsg)
		}
		if e.Err != wantErr {
			t.Errorf("Err = %v, want %v", e.Err, wantErr)
		}
		if e.HTTPStatus != wantStatus {
			t.Errorf("HTTPStatus = %v, want %v", e.HTTPStatus, wantStatus)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.factory()
			checkAppError(t, e, tt.wantLayer, tt.wantCode, tt.wantMsg, tt.wantErr, tt.wantStatus)
		})
	}
}

func TestToHTTPErrorFunction(t *testing.T) {
	internalErr := NewDomainError("D_ERR", "domain failure", nil, 400)

	// Caso erro seja AppError
	resp := ToHTTPError(internalErr)
	if resp.Code != "D_ERR" || resp.Message != "domain failure" {
		t.Errorf("ToHTTPError(AppError) retornou %+v, esperado código D_ERR", resp)
	}

	// Caso erro genérico (não AppError)
	otherErr := errors.New("random error")
	resp2 := ToHTTPError(otherErr)
	if resp2.Code != "INTERNAL_ERROR" || resp2.Message != "internal error" {
		t.Errorf("ToHTTPError(error) retornou %+v, esperado código INTERNAL_ERROR", resp2)
	}
}
