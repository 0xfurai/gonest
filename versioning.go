package gonest

import (
	"net/http"
	"strings"
)

// VersioningType determines how the API version is extracted.
type VersioningType int

const (
	// VersioningURI extracts version from the URL path (e.g., /v1/cats).
	VersioningURI VersioningType = iota
	// VersioningHeader extracts version from a request header.
	VersioningHeader
	// VersioningMediaType extracts version from the Accept header media type.
	VersioningMediaType
	// VersioningCustom uses a user-provided extraction function.
	VersioningCustom
)

// VersionExtractor is a function that extracts the API version from a request context.
// Used with VersioningCustom.
type VersionExtractor func(ctx Context) string

// VersioningOptions configures API versioning.
type VersioningOptions struct {
	Type           VersioningType
	Header         string // header name for VersioningHeader (default: "X-API-Version")
	DefaultVersion string
	Extractor      VersionExtractor // custom extractor for VersioningCustom
}

// VersionNeutral is a sentinel value indicating a route handles all versions.
const VersionNeutral = "__VERSION_NEUTRAL__"

// VersioningMiddleware extracts the API version and stores it in context.
type VersioningMiddleware struct {
	opts VersioningOptions
}

// NewVersioningMiddleware creates versioning middleware.
func NewVersioningMiddleware(opts VersioningOptions) *VersioningMiddleware {
	if opts.Header == "" {
		opts.Header = "X-API-Version"
	}
	return &VersioningMiddleware{opts: opts}
}

func (m *VersioningMiddleware) Use(ctx Context, next NextFunc) error {
	version := ""

	switch m.opts.Type {
	case VersioningURI:
		version = extractURIVersion(ctx.Path())
	case VersioningHeader:
		version = ctx.Header(m.opts.Header)
	case VersioningMediaType:
		version = extractMediaTypeVersion(ctx.Header("Accept"))
	case VersioningCustom:
		if m.opts.Extractor != nil {
			version = m.opts.Extractor(ctx)
		}
	}

	if version == "" {
		version = m.opts.DefaultVersion
	}

	ctx.Set("__api_version", version)
	return next()
}

// GetVersion retrieves the API version from the context.
func GetVersion(ctx Context) string {
	v, ok := ctx.Get("__api_version")
	if !ok {
		return ""
	}
	return v.(string)
}

// VersionGuard restricts a route to specific API versions.
// Set version via route metadata: .SetMetadata("version", "1")
type VersionGuard struct{}

func NewVersionGuard() *VersionGuard {
	return &VersionGuard{}
}

func (g *VersionGuard) CanActivate(ctx ExecutionContext) (bool, error) {
	requiredVersion, ok := GetMetadata[string](ctx, "version")
	if !ok {
		return true, nil // no version constraint
	}

	// VersionNeutral routes match all versions.
	if requiredVersion == VersionNeutral {
		return true, nil
	}

	currentVersion := GetVersion(ctx)
	if currentVersion == "" {
		return true, nil // no version in request, allow
	}

	if currentVersion != requiredVersion {
		return false, NewHTTPException(http.StatusNotFound,
			"Version "+currentVersion+" does not support this endpoint")
	}
	return true, nil
}

func extractURIVersion(path string) string {
	// Extract /v1/... or /v2/...
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	if len(parts) > 0 && len(parts[0]) > 1 && parts[0][0] == 'v' {
		return parts[0][1:]
	}
	return ""
}

func extractMediaTypeVersion(accept string) string {
	// Accept: application/json;v=2
	parts := strings.Split(accept, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "v=") {
			return part[2:]
		}
	}
	return ""
}
