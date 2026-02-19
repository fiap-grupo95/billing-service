package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Layer identifica a camada da aplicação onde o erro ocorreu
type Layer string

const (
	LayerDomain         Layer = "domain"
	LayerApplication    Layer = "application"
	LayerInfrastructure Layer = "infrastructure"
)

// ErrorResponse é a estrutura para enviar erros para o cliente HTTP
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// AppError é o erro customizado do sistema, com info estruturada
type AppError struct {
	Code       string // Código identificador único do erro
	Message    string // Mensagem amigável para o usuário
	Layer      Layer  // Camada onde ocorreu o erro
	Err        error  // Erro interno original (para logs)
	HTTPStatus int    // Código HTTP associado (ex: 400, 404, 500)
}

// Error implementa a interface error para o AppError
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Layer: %s, Code: %s, Message: %s, Internal Error: %v",
			e.Layer, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Layer, e.Code, e.Message)
}

// Unwrap permite usar errors.Unwrap para obter o erro interno
func (e *AppError) Unwrap() error {
	return e.Err
}

// ToHTTPError transforma o AppError em um ErrorResponse para API
func (e *AppError) ToHTTPError() ErrorResponse {
	return ErrorResponse{Code: e.Code, Message: e.Message}
}

// ToJSON retorna o JSON serializado do ErrorResponse (útil para API)
func (e *AppError) ToJSON() []byte {
	resp := e.ToHTTPError()
	b, _ := json.Marshal(resp)
	return b
}

// Factory functions para criar erros em diferentes camadas

func NewDomainError(code, msg string, err error, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    msg,
		Layer:      LayerDomain,
		Err:        err,
		HTTPStatus: httpStatus,
	}
}

// Simplificado quando não tem erro interno
func NewDomainErrorSimple(code, msg string, httpStatus int) *AppError {
	return NewDomainError(code, msg, nil, httpStatus)
}

func NewApplicationError(code, msg string, err error, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    msg,
		Layer:      LayerApplication,
		Err:        err,
		HTTPStatus: httpStatus,
	}
}

func NewInfraError(code, msg string, err error, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    msg,
		Layer:      LayerInfrastructure,
		Err:        err,
		HTTPStatus: httpStatus,
	}
}

// ToHTTPError é a função que recebe qualquer error e tenta converter para AppError
// Caso não consiga, retorna erro genérico para API
func ToHTTPError(err error) ErrorResponse {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.ToHTTPError()
	}
	return ErrorResponse{Code: "INTERNAL_ERROR", Message: "internal error"}
}
