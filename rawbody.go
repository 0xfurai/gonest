package gonest

import (
	"bytes"
	"io"
)

// RawBody retrieves the raw request body as a byte slice.
// Equivalent to NestJS @RawBody() decorator.
// The body is buffered on first access and cached in the context so it can be
// read multiple times (unlike the default io.ReadCloser which is single-read).
func RawBody(ctx Context) ([]byte, error) {
	// Check if already cached
	if cached, ok := ctx.Get("__raw_body"); ok {
		return cached.([]byte), nil
	}

	body := ctx.Body()
	if body == nil {
		return nil, NewBadRequestException("empty request body")
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, NewBadRequestException("failed to read body: " + err.Error())
	}
	body.Close()

	// Cache for subsequent reads
	ctx.Set("__raw_body", data)

	// Replace the body so Bind() can still work after RawBody()
	ctx.Request().Body = io.NopCloser(bytes.NewReader(data))

	return data, nil
}

// RawBodyMiddleware pre-reads the request body and caches it so that both
// RawBody() and Bind() work in any order. Apply this middleware globally or
// on specific routes that need raw body access.
type RawBodyMiddleware struct{}

func NewRawBodyMiddleware() *RawBodyMiddleware {
	return &RawBodyMiddleware{}
}

func (m *RawBodyMiddleware) Use(ctx Context, next NextFunc) error {
	r := ctx.Request()
	if r.Body != nil && r.ContentLength != 0 {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return NewBadRequestException("failed to read body: " + err.Error())
		}
		r.Body.Close()
		ctx.Set("__raw_body", data)
		r.Body = io.NopCloser(bytes.NewReader(data))
	}
	return next()
}

