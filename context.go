package gonest

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"
)

// HandlerFunc is the signature for route handlers.
type HandlerFunc func(ctx Context) error

// NextFunc advances to the next handler in the chain.
type NextFunc func() error

// Context provides request/response access to route handlers.
type Context interface {
	// Request returns the underlying *http.Request.
	Request() *http.Request
	// ResponseWriter returns the underlying http.ResponseWriter.
	ResponseWriter() http.ResponseWriter
	// Param returns a path parameter value, possibly transformed by pipes.
	Param(name string) any
	// Query returns a query string parameter.
	Query(name string) string
	// Header returns a request header value.
	Header(name string) string
	// Bind decodes the request body into the given struct (JSON).
	Bind(v any) error
	// JSON writes a JSON response with the given status code.
	JSON(statusCode int, v any) error
	// String writes a plain text response.
	String(statusCode int, s string) error
	// Status sets the response status code and returns the context for chaining.
	Status(code int) Context
	// NoContent sends a response with no body.
	NoContent(statusCode int) error
	// Redirect sends an HTTP redirect.
	Redirect(statusCode int, url string) error
	// Set stores a value in the request-scoped store.
	Set(key string, value any)
	// Get retrieves a value from the request-scoped store.
	Get(key string) (any, bool)
	// Ctx returns the request's context.Context.
	Ctx() context.Context
	// SetHeader sets a response header.
	SetHeader(key, value string)
	// Cookie returns a named cookie from the request.
	Cookie(name string) (*http.Cookie, error)
	// SetCookie adds a Set-Cookie header to the response.
	SetCookie(cookie *http.Cookie)
	// FormFile returns the first file for the given form key.
	FormFile(name string) (multipart.File, *multipart.FileHeader, error)
	// IP returns the client's IP address.
	IP() string
	// Path returns the matched route path.
	Path() string
	// Method returns the HTTP method.
	Method() string
	// QueryValues returns all query parameters.
	QueryValues() url.Values
	// Body returns the raw request body reader.
	Body() io.ReadCloser
	// Written reports whether the response has been written.
	Written() bool
}

// defaultContext is the standard Context implementation.
type defaultContext struct {
	req     *http.Request
	writer  http.ResponseWriter
	params  map[string]any
	store   map[string]any
	mu      sync.RWMutex
	written bool
	status  int
}

func newContext(w http.ResponseWriter, r *http.Request) *defaultContext {
	return &defaultContext{
		req:    r,
		writer: w,
		params: make(map[string]any),
		store:  make(map[string]any),
	}
}

func (c *defaultContext) Request() *http.Request           { return c.req }
func (c *defaultContext) ResponseWriter() http.ResponseWriter { return c.writer }

func (c *defaultContext) Param(name string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.params[name]
}

func (c *defaultContext) setParam(name string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.params[name] = value
}

func (c *defaultContext) Query(name string) string {
	return c.req.URL.Query().Get(name)
}

func (c *defaultContext) Header(name string) string {
	return c.req.Header.Get(name)
}

func (c *defaultContext) Bind(v any) error {
	if c.req.Body == nil {
		return NewBadRequestException("empty request body")
	}
	defer c.req.Body.Close()
	if err := json.NewDecoder(c.req.Body).Decode(v); err != nil {
		return NewBadRequestException("invalid JSON: " + err.Error())
	}
	return nil
}

func (c *defaultContext) JSON(statusCode int, v any) error {
	c.writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.writer.WriteHeader(statusCode)
	c.written = true
	if v == nil {
		return nil
	}
	return json.NewEncoder(c.writer).Encode(v)
}

func (c *defaultContext) String(statusCode int, s string) error {
	c.writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.writer.WriteHeader(statusCode)
	c.written = true
	_, err := c.writer.Write([]byte(s))
	return err
}

func (c *defaultContext) Status(code int) Context {
	c.status = code
	return c
}

func (c *defaultContext) NoContent(statusCode int) error {
	c.writer.WriteHeader(statusCode)
	c.written = true
	return nil
}

func (c *defaultContext) Redirect(statusCode int, url string) error {
	http.Redirect(c.writer, c.req, url, statusCode)
	c.written = true
	return nil
}

func (c *defaultContext) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
}

func (c *defaultContext) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.store[key]
	return v, ok
}

func (c *defaultContext) Ctx() context.Context { return c.req.Context() }

func (c *defaultContext) SetHeader(key, value string) {
	c.writer.Header().Set(key, value)
}

func (c *defaultContext) Cookie(name string) (*http.Cookie, error) {
	return c.req.Cookie(name)
}

func (c *defaultContext) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.writer, cookie)
}

func (c *defaultContext) FormFile(name string) (multipart.File, *multipart.FileHeader, error) {
	return c.req.FormFile(name)
}

func (c *defaultContext) IP() string {
	if ip := c.req.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := c.req.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return c.req.RemoteAddr
}

func (c *defaultContext) Path() string        { return c.req.URL.Path }
func (c *defaultContext) Method() string      { return c.req.Method }
func (c *defaultContext) QueryValues() url.Values { return c.req.URL.Query() }
func (c *defaultContext) Body() io.ReadCloser { return c.req.Body }
func (c *defaultContext) Written() bool       { return c.written }

// ExecutionContext extends Context with metadata about the currently executing
// handler. It is passed to guards, interceptors, and pipes.
type ExecutionContext interface {
	Context
	// GetHandler returns an identifier for the current route handler.
	GetHandler() any
	// GetClass returns an identifier for the current controller.
	GetClass() any
	// GetMetadata returns route-level metadata by key.
	GetMetadata(key string) (any, bool)
	// SwitchToHTTP returns the HTTP-specific context.
	SwitchToHTTP() HTTPContext
	// GetType returns the execution context type ("http", "ws", "rpc").
	GetType() string
}

// HTTPContext provides typed access to HTTP request/response.
type HTTPContext interface {
	Request() *http.Request
	Response() ResponseWriter
}

// ResponseWriter extends http.ResponseWriter with helper methods.
type ResponseWriter interface {
	http.ResponseWriter
	Status(code int) ResponseWriter
	JSON(v any) error
}

// executionContext is the default ExecutionContext implementation.
type executionContext struct {
	Context
	handler    any
	controller any
	metadata   map[string]any
}

func newExecutionContext(ctx Context, handler any, controller any, metadata map[string]any) *executionContext {
	return &executionContext{
		Context:    ctx,
		handler:    handler,
		controller: controller,
		metadata:   metadata,
	}
}

func (e *executionContext) GetHandler() any  { return e.handler }
func (e *executionContext) GetClass() any    { return e.controller }
func (e *executionContext) GetType() string  { return "http" }

func (e *executionContext) GetMetadata(key string) (any, bool) {
	v, ok := e.metadata[key]
	return v, ok
}

func (e *executionContext) SwitchToHTTP() HTTPContext {
	return &httpContext{ctx: e}
}

type httpContext struct {
	ctx *executionContext
}

func (h *httpContext) Request() *http.Request { return h.ctx.Request() }
func (h *httpContext) Response() ResponseWriter {
	return &responseWriter{w: h.ctx.ResponseWriter()}
}

type responseWriter struct {
	w          http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) Header() http.Header         { return rw.w.Header() }
func (rw *responseWriter) Write(b []byte) (int, error) { return rw.w.Write(b) }
func (rw *responseWriter) WriteHeader(code int)         { rw.w.WriteHeader(code) }

func (rw *responseWriter) Status(code int) ResponseWriter {
	rw.statusCode = code
	return rw
}

func (rw *responseWriter) JSON(v any) error {
	rw.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if rw.statusCode > 0 {
		rw.w.WriteHeader(rw.statusCode)
	}
	if v == nil {
		return nil
	}
	return json.NewEncoder(rw.w).Encode(v)
}

// ArgumentsHost provides access to the underlying transport layer.
// Used by exception filters.
type ArgumentsHost interface {
	SwitchToHTTP() HTTPArgumentsHost
	GetType() string
}

// HTTPArgumentsHost provides HTTP-specific access for exception filters.
type HTTPArgumentsHost interface {
	Request() *http.Request
	Response() ResponseWriter
}

type argumentsHost struct {
	ctx Context
}

func newArgumentsHost(ctx Context) ArgumentsHost {
	return &argumentsHost{ctx: ctx}
}

func (h *argumentsHost) GetType() string { return "http" }
func (h *argumentsHost) SwitchToHTTP() HTTPArgumentsHost {
	return &httpArgumentsHost{ctx: h.ctx}
}

type httpArgumentsHost struct {
	ctx Context
}

func (h *httpArgumentsHost) Request() *http.Request { return h.ctx.Request() }
func (h *httpArgumentsHost) Response() ResponseWriter {
	return &responseWriter{w: h.ctx.ResponseWriter()}
}

// CallHandler is passed to interceptors to invoke the next handler in the chain.
type CallHandler struct {
	fn func() (any, error)
}

func NewCallHandler(fn func() (any, error)) CallHandler {
	return CallHandler{fn: fn}
}

func (c CallHandler) Handle() (any, error) {
	return c.fn()
}
