package gonest

import (
	"net/http"
	"testing"
)

func TestRouter_RegisterRoutes(t *testing.T) {
	r := newRouter()
	r.Prefix("/cats")
	r.Get("/", func(ctx Context) error { return nil })
	r.Post("/", func(ctx Context) error { return nil })
	r.Get("/:id", func(ctx Context) error { return nil })

	routes := r.resolvedRoutes()
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	if routes[0].Method != http.MethodGet || routes[0].Path != "/cats/" {
		t.Errorf("unexpected route 0: %s %s", routes[0].Method, routes[0].Path)
	}
	if routes[1].Method != http.MethodPost || routes[1].Path != "/cats/" {
		t.Errorf("unexpected route 1: %s %s", routes[1].Method, routes[1].Path)
	}
	if routes[2].Method != http.MethodGet || routes[2].Path != "/cats/:id" {
		t.Errorf("unexpected route 2: %s %s", routes[2].Method, routes[2].Path)
	}
}

func TestRouter_AllMethods(t *testing.T) {
	r := newRouter()
	r.Get("/get", func(ctx Context) error { return nil })
	r.Post("/post", func(ctx Context) error { return nil })
	r.Put("/put", func(ctx Context) error { return nil })
	r.Delete("/delete", func(ctx Context) error { return nil })
	r.Patch("/patch", func(ctx Context) error { return nil })
	r.Options("/options", func(ctx Context) error { return nil })
	r.Head("/head", func(ctx Context) error { return nil })
	r.All("/all", func(ctx Context) error { return nil })

	routes := r.resolvedRoutes()
	if len(routes) != 8 {
		t.Fatalf("expected 8 routes, got %d", len(routes))
	}

	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "*"}
	for i, route := range routes {
		if route.Method != expectedMethods[i] {
			t.Errorf("route %d: expected method %q, got %q", i, expectedMethods[i], route.Method)
		}
	}
}

func TestRouteBuilder_SetMetadata(t *testing.T) {
	r := newRouter()
	rb := r.Get("/test", func(ctx Context) error { return nil })
	rb.SetMetadata("roles", []string{"admin"})

	routes := r.resolvedRoutes()
	roles, ok := routes[0].Metadata["roles"]
	if !ok {
		t.Fatal("expected metadata")
	}
	if len(roles.([]string)) != 1 {
		t.Errorf("expected 1 role, got %v", roles)
	}
}

func TestRouteBuilder_Pipes(t *testing.T) {
	r := newRouter()
	pipe := NewParseIntPipe("id")
	rb := r.Get("/:id", func(ctx Context) error { return nil })
	rb.Pipes(pipe)

	routes := r.resolvedRoutes()
	if len(routes[0].Pipes) != 1 {
		t.Errorf("expected 1 pipe, got %d", len(routes[0].Pipes))
	}
}

func TestRouteBuilder_Guards(t *testing.T) {
	r := newRouter()
	guard := GuardFunc(func(ctx ExecutionContext) (bool, error) { return true, nil })
	rb := r.Get("/test", func(ctx Context) error { return nil })
	rb.Guards(guard)

	routes := r.resolvedRoutes()
	if len(routes[0].Guards) != 1 {
		t.Errorf("expected 1 guard, got %d", len(routes[0].Guards))
	}
}

func TestRouteBuilder_HttpCode(t *testing.T) {
	r := newRouter()
	rb := r.Post("/test", func(ctx Context) error { return nil })
	rb.HttpCode(201)

	routes := r.resolvedRoutes()
	code, ok := routes[0].Metadata["__httpCode"]
	if !ok || code != 201 {
		t.Errorf("expected httpCode 201, got %v", code)
	}
}

func TestRouteBuilder_Header(t *testing.T) {
	r := newRouter()
	rb := r.Get("/test", func(ctx Context) error { return nil })
	rb.Header("X-Custom", "value1").Header("X-Other", "value2")

	routes := r.resolvedRoutes()
	headers, ok := routes[0].Metadata["__headers"].([][2]string)
	if !ok || len(headers) != 2 {
		t.Errorf("expected 2 headers, got %v", routes[0].Metadata["__headers"])
	}
}

func TestRouter_ControllerLevelGuards(t *testing.T) {
	r := newRouter()
	ctrlGuard := GuardFunc(func(ctx ExecutionContext) (bool, error) { return true, nil })
	routeGuard := GuardFunc(func(ctx ExecutionContext) (bool, error) { return false, nil })

	r.UseGuards(ctrlGuard)
	r.Get("/test", func(ctx Context) error { return nil }).Guards(routeGuard)

	routes := r.resolvedRoutes()
	if len(routes[0].Guards) != 2 {
		t.Errorf("expected 2 guards (controller + route), got %d", len(routes[0].Guards))
	}
}

func TestRouter_ControllerLevelInterceptors(t *testing.T) {
	r := newRouter()
	interceptor := InterceptorFunc(func(ctx ExecutionContext, next CallHandler) (any, error) {
		return next.Handle()
	})
	r.UseInterceptors(interceptor)
	r.Get("/test", func(ctx Context) error { return nil })

	routes := r.resolvedRoutes()
	if len(routes[0].Interceptors) != 1 {
		t.Errorf("expected 1 interceptor, got %d", len(routes[0].Interceptors))
	}
}

func TestRouteBuilder_Redirect(t *testing.T) {
	r := newRouter()
	rb := r.Get("/old", func(ctx Context) error { return nil })
	rb.Redirect("/new", 301)

	routes := r.resolvedRoutes()
	redirect, ok := routes[0].Metadata["__redirect"]
	if !ok {
		t.Fatal("expected redirect metadata")
	}
	arr := redirect.([2]any)
	if arr[0] != "/new" || arr[1] != 301 {
		t.Errorf("unexpected redirect: %v", arr)
	}
}
