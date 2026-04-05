package common

import (
	"fmt"
	"time"

	"github.com/0xfurai/gonest"
)

// RequestLoggerMiddleware logs every request with timing.
type RequestLoggerMiddleware struct {
	logger gonest.Logger
}

func NewRequestLoggerMiddleware(logger gonest.Logger) *RequestLoggerMiddleware {
	return &RequestLoggerMiddleware{logger: logger}
}

func (m *RequestLoggerMiddleware) Use(ctx gonest.Context, next gonest.NextFunc) error {
	start := time.Now()
	err := next()
	duration := time.Since(start)

	status := 200
	if err != nil {
		if httpErr, ok := err.(*gonest.HTTPException); ok {
			status = httpErr.StatusCode()
		} else {
			status = 500
		}
	}

	m.logger.Log("%s %s %d %v", ctx.Method(), ctx.Path(), status, duration)
	return err
}

// RequestIDMiddleware adds a unique request ID to each request.
type RequestIDMiddleware struct {
	counter int
}

func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

func (m *RequestIDMiddleware) Use(ctx gonest.Context, next gonest.NextFunc) error {
	m.counter++
	requestID := fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), m.counter)
	ctx.SetHeader("X-Request-ID", requestID)
	ctx.Set("requestId", requestID)
	return next()
}
