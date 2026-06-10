package apierror

import "net/http"

type ErrorCode int

const (
	CodeBadRequest     ErrorCode = 40000
	CodeUnauthorized   ErrorCode = 40100
	CodeForbidden      ErrorCode = 40300
	CodeNotFound       ErrorCode = 40400
	CodeRateLimited    ErrorCode = 42900
	CodeInternalError  ErrorCode = 50000
	CodeProviderError  ErrorCode = 50200
	CodeQuotaExceeded  ErrorCode = 40301
)

type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	HTTP    int       `json:"-"`
}

func (e *APIError) Error() string {
	return e.Message
}

func BadRequest(msg string) *APIError {
	return &APIError{Code: CodeBadRequest, Message: msg, HTTP: http.StatusBadRequest}
}

func Unauthorized(msg string) *APIError {
	return &APIError{Code: CodeUnauthorized, Message: msg, HTTP: http.StatusUnauthorized}
}

func Forbidden(msg string) *APIError {
	return &APIError{Code: CodeForbidden, Message: msg, HTTP: http.StatusForbidden}
}

func NotFound(msg string) *APIError {
	return &APIError{Code: CodeNotFound, Message: msg, HTTP: http.StatusNotFound}
}

func RateLimited(msg string) *APIError {
	return &APIError{Code: CodeRateLimited, Message: msg, HTTP: http.StatusTooManyRequests}
}

func InternalError(msg string) *APIError {
	return &APIError{Code: CodeInternalError, Message: msg, HTTP: http.StatusInternalServerError}
}

func ProviderError(msg string) *APIError {
	return &APIError{Code: CodeProviderError, Message: msg, HTTP: http.StatusBadGateway}
}

func QuotaExceeded(msg string) *APIError {
	return &APIError{Code: CodeQuotaExceeded, Message: msg, HTTP: http.StatusForbidden}
}
