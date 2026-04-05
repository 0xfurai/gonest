package gonest

import (
	"fmt"
	"net/http"
)

// HTTPException is the base error type for all HTTP errors in the framework.
type HTTPException struct {
	statusCode int
	message    string
	cause      error
}

func (e *HTTPException) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.message, e.cause.Error())
	}
	return e.message
}

func (e *HTTPException) StatusCode() int { return e.statusCode }
func (e *HTTPException) Cause() error    { return e.cause }

func (e *HTTPException) Unwrap() error { return e.cause }

func NewHTTPException(statusCode int, message string) *HTTPException {
	return &HTTPException{statusCode: statusCode, message: message}
}

func WrapHTTPException(statusCode int, message string, cause error) *HTTPException {
	return &HTTPException{statusCode: statusCode, message: message, cause: cause}
}

func NewBadRequestException(message string) *HTTPException {
	return NewHTTPException(http.StatusBadRequest, message)
}

func NewUnauthorizedException(message string) *HTTPException {
	return NewHTTPException(http.StatusUnauthorized, message)
}

func NewForbiddenException(message string) *HTTPException {
	return NewHTTPException(http.StatusForbidden, message)
}

func NewNotFoundException(message string) *HTTPException {
	return NewHTTPException(http.StatusNotFound, message)
}

func NewConflictException(message string) *HTTPException {
	return NewHTTPException(http.StatusConflict, message)
}

func NewGoneException(message string) *HTTPException {
	return NewHTTPException(http.StatusGone, message)
}

func NewUnprocessableEntityException(message string) *HTTPException {
	return NewHTTPException(http.StatusUnprocessableEntity, message)
}

func NewInternalServerError(message string) *HTTPException {
	return NewHTTPException(http.StatusInternalServerError, message)
}

func NewNotImplementedException(message string) *HTTPException {
	return NewHTTPException(http.StatusNotImplemented, message)
}

func NewBadGatewayException(message string) *HTTPException {
	return NewHTTPException(http.StatusBadGateway, message)
}

func NewServiceUnavailableException(message string) *HTTPException {
	return NewHTTPException(http.StatusServiceUnavailable, message)
}

func NewMethodNotAllowedException(message string) *HTTPException {
	return NewHTTPException(http.StatusMethodNotAllowed, message)
}

func NewRequestTimeoutException(message string) *HTTPException {
	return NewHTTPException(http.StatusRequestTimeout, message)
}

func NewPayloadTooLargeException(message string) *HTTPException {
	return NewHTTPException(http.StatusRequestEntityTooLarge, message)
}

func NewUnsupportedMediaTypeException(message string) *HTTPException {
	return NewHTTPException(http.StatusUnsupportedMediaType, message)
}

func NewTooManyRequestsException(message string) *HTTPException {
	return NewHTTPException(http.StatusTooManyRequests, message)
}

func NewNotAcceptableException(message string) *HTTPException {
	return NewHTTPException(http.StatusNotAcceptable, message)
}

func NewPreconditionFailedException(message string) *HTTPException {
	return NewHTTPException(http.StatusPreconditionFailed, message)
}

func NewImATeapotException(message string) *HTTPException {
	return NewHTTPException(http.StatusTeapot, message)
}

func NewMisdirectedException(message string) *HTTPException {
	return NewHTTPException(http.StatusMisdirectedRequest, message)
}

func NewGatewayTimeoutException(message string) *HTTPException {
	return NewHTTPException(http.StatusGatewayTimeout, message)
}

func NewHttpVersionNotSupportedException(message string) *HTTPException {
	return NewHTTPException(http.StatusHTTPVersionNotSupported, message)
}
