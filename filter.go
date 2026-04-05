package gonest

import (
	"encoding/json"
	"net/http"
	"time"
)

// ExceptionFilter catches errors thrown during request processing and
// produces an appropriate response. Equivalent to NestJS ExceptionFilter.
type ExceptionFilter interface {
	Catch(err error, host ArgumentsHost) error
}

// ExceptionFilterFunc is a convenience adapter for simple filter functions.
type ExceptionFilterFunc func(err error, host ArgumentsHost) error

func (f ExceptionFilterFunc) Catch(err error, host ArgumentsHost) error {
	return f(err, host)
}

// DefaultExceptionFilter is the built-in exception filter that handles
// HTTPException errors and produces a standard JSON error response.
type DefaultExceptionFilter struct{}

func (f *DefaultExceptionFilter) Catch(err error, host ArgumentsHost) error {
	httpCtx := host.SwitchToHTTP()
	resp := httpCtx.Response()
	req := httpCtx.Request()

	statusCode := http.StatusInternalServerError
	message := "Internal Server Error"

	if httpErr, ok := err.(*HTTPException); ok {
		statusCode = httpErr.StatusCode()
		message = httpErr.Error()
	}

	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.WriteHeader(statusCode)
	return json.NewEncoder(resp).Encode(map[string]any{
		"statusCode": statusCode,
		"message":    message,
		"timestamp":  time.Now().Format(time.RFC3339),
		"path":       req.URL.Path,
	})
}
