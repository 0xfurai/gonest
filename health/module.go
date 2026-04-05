package health

import (
	"net/http"
	"sync"

	"github.com/0xfurai/gonest"
)

// Status represents the health status of a component.
type Status string

const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

// HealthIndicator checks the health of a component.
type HealthIndicator interface {
	// Name returns the indicator name (e.g., "database", "redis").
	Name() string
	// Check returns the health status and optional details.
	Check() HealthResult
}

// HealthResult is the result of a single health check.
type HealthResult struct {
	Status  Status         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}

// HealthResponse is the full health check response.
type HealthResponse struct {
	Status    Status                   `json:"status"`
	Info      map[string]HealthResult  `json:"info,omitempty"`
	Error     map[string]HealthResult  `json:"error,omitempty"`
	Details   map[string]HealthResult  `json:"details,omitempty"`
}

// HealthService manages health indicators and produces overall health status.
type HealthService struct {
	mu         sync.RWMutex
	indicators []HealthIndicator
}

// NewHealthService creates a new health service.
func NewHealthService() *HealthService {
	return &HealthService{}
}

// AddIndicator registers a health indicator.
func (s *HealthService) AddIndicator(indicator HealthIndicator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.indicators = append(s.indicators, indicator)
}

// Check runs all health indicators and returns the overall status.
func (s *HealthService) Check() HealthResponse {
	s.mu.RLock()
	indicators := make([]HealthIndicator, len(s.indicators))
	copy(indicators, s.indicators)
	s.mu.RUnlock()

	resp := HealthResponse{
		Status:  StatusUp,
		Info:    make(map[string]HealthResult),
		Error:   make(map[string]HealthResult),
		Details: make(map[string]HealthResult),
	}

	for _, ind := range indicators {
		result := ind.Check()
		resp.Details[ind.Name()] = result
		if result.Status == StatusUp {
			resp.Info[ind.Name()] = result
		} else {
			resp.Error[ind.Name()] = result
			resp.Status = StatusDown
		}
	}

	return resp
}

// healthController exposes the /health endpoint.
type healthController struct {
	service *HealthService
	path    string
}

func (c *healthController) Register(r gonest.Router) {
	// Health check is always public
	r.Get(c.path, c.check).SetMetadata("public", true)
}

func (c *healthController) check(ctx gonest.Context) error {
	resp := c.service.Check()
	statusCode := http.StatusOK
	if resp.Status == StatusDown {
		statusCode = http.StatusServiceUnavailable
	}
	return ctx.JSON(statusCode, resp)
}

// Options configures the health module.
type Options struct {
	// Path is the health check endpoint (default: "/health").
	Path string
	// Indicators are the health indicators to register.
	Indicators []HealthIndicator
}

// NewModule creates a health check module.
func NewModule(opts ...Options) *gonest.Module {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Path == "" {
		opt.Path = "/health"
	}

	svc := NewHealthService()
	for _, ind := range opt.Indicators {
		svc.AddIndicator(ind)
	}

	ctrl := &healthController{service: svc, path: opt.Path}

	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *healthController { return ctrl }},
		Providers:   []any{gonest.ProvideValue[*HealthService](svc)},
		Exports:     []any{(*HealthService)(nil)},
	})
}

// --- Built-in Indicators ---

// PingIndicator always returns "up". Useful for basic liveness checks.
type PingIndicator struct{}

func (p *PingIndicator) Name() string        { return "ping" }
func (p *PingIndicator) Check() HealthResult { return HealthResult{Status: StatusUp} }

// CustomIndicator wraps a check function as a HealthIndicator.
type CustomIndicator struct {
	IndicatorName string
	CheckFn       func() HealthResult
}

func (c *CustomIndicator) Name() string        { return c.IndicatorName }
func (c *CustomIndicator) Check() HealthResult { return c.CheckFn() }
