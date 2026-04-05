package gonest

import (
	"errors"
	"net/http"
	"testing"
)

func TestHTTPException_Error(t *testing.T) {
	ex := NewBadRequestException("invalid input")
	if ex.Error() != "invalid input" {
		t.Errorf("expected 'invalid input', got %q", ex.Error())
	}
	if ex.StatusCode() != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, ex.StatusCode())
	}
}

func TestHTTPException_WithCause(t *testing.T) {
	cause := errors.New("root cause")
	ex := WrapHTTPException(http.StatusInternalServerError, "wrapped", cause)
	if ex.Error() != "wrapped: root cause" {
		t.Errorf("expected 'wrapped: root cause', got %q", ex.Error())
	}
	if !errors.Is(ex, cause) {
		t.Error("expected errors.Is to find cause")
	}
}

func TestHTTPException_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) *HTTPException
		expected int
	}{
		{"BadRequest", NewBadRequestException, 400},
		{"Unauthorized", NewUnauthorizedException, 401},
		{"Forbidden", NewForbiddenException, 403},
		{"NotFound", NewNotFoundException, 404},
		{"Conflict", NewConflictException, 409},
		{"Gone", NewGoneException, 410},
		{"UnprocessableEntity", NewUnprocessableEntityException, 422},
		{"InternalServerError", NewInternalServerError, 500},
		{"NotImplemented", NewNotImplementedException, 501},
		{"BadGateway", NewBadGatewayException, 502},
		{"ServiceUnavailable", NewServiceUnavailableException, 503},
		{"MethodNotAllowed", NewMethodNotAllowedException, 405},
		{"RequestTimeout", NewRequestTimeoutException, 408},
		{"PayloadTooLarge", NewPayloadTooLargeException, 413},
		{"UnsupportedMediaType", NewUnsupportedMediaTypeException, 415},
		{"TooManyRequests", NewTooManyRequestsException, 429},
		{"NotAcceptable", NewNotAcceptableException, 406},
		{"PreconditionFailed", NewPreconditionFailedException, 412},
		{"ImATeapot", NewImATeapotException, 418},
		{"Misdirected", NewMisdirectedException, 421},
		{"GatewayTimeout", NewGatewayTimeoutException, 504},
		{"HttpVersionNotSupported", NewHttpVersionNotSupportedException, 505},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex := tt.fn("test")
			if ex.StatusCode() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, ex.StatusCode())
			}
			if ex.Error() != "test" {
				t.Errorf("expected 'test', got %q", ex.Error())
			}
		})
	}
}
