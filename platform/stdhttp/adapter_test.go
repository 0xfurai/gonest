package stdhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonest/platform"
)

func TestAdapter_BasicRouting(t *testing.T) {
	a := New()
	a.Handle("GET", "/hello", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.WriteHeader(200)
		w.Write([]byte("hello"))
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("expected 'hello', got %q", w.Body.String())
	}
}

func TestAdapter_PathParams(t *testing.T) {
	a := New()
	a.Handle("GET", "/users/:id", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Write([]byte("user:" + params["id"]))
	})

	req := httptest.NewRequest("GET", "/users/42", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Body.String() != "user:42" {
		t.Errorf("expected 'user:42', got %q", w.Body.String())
	}
}

func TestAdapter_MultipleParams(t *testing.T) {
	a := New()
	a.Handle("GET", "/users/:userId/posts/:postId", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Write([]byte(params["userId"] + ":" + params["postId"]))
	})

	req := httptest.NewRequest("GET", "/users/1/posts/99", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Body.String() != "1:99" {
		t.Errorf("expected '1:99', got %q", w.Body.String())
	}
}

func TestAdapter_NotFound(t *testing.T) {
	a := New()
	a.Handle("GET", "/exists", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAdapter_MethodNotAllowed(t *testing.T) {
	a := New()
	a.Handle("GET", "/test", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Code != 405 {
		t.Errorf("expected 405, got %d", w.Code)
	}
	if w.Header().Get("Allow") == "" {
		t.Error("expected Allow header")
	}
}

func TestAdapter_WildcardRoute(t *testing.T) {
	a := New()
	a.Handle("GET", "/static/*", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Write([]byte("static"))
	})

	req := httptest.NewRequest("GET", "/static/css/main.css", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Body.String() != "static" {
		t.Errorf("expected 'static', got %q", w.Body.String())
	}
}

func TestAdapter_RootRoute(t *testing.T) {
	a := New()
	a.Handle("GET", "/", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Write([]byte("root"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Body.String() != "root" {
		t.Errorf("expected 'root', got %q", w.Body.String())
	}
}

func TestAdapter_AllMethods(t *testing.T) {
	a := New()
	a.Handle("*", "/test", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Write([]byte(r.Method))
	})

	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		req := httptest.NewRequest(method, "/test", nil)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)

		if w.Body.String() != method {
			t.Errorf("for %s: expected %q, got %q", method, method, w.Body.String())
		}
	}
}

func TestAdapter_Middleware(t *testing.T) {
	a := New()
	a.Handle("GET", "/test", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		w.Write([]byte("ok"))
	})
	a.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)

	if w.Header().Get("X-Middleware") != "applied" {
		t.Error("expected middleware header")
	}
}

func TestAdapter_CustomNotFoundHandler(t *testing.T) {
	a := New()
	a.SetNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("custom 404"))
	})

	req := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if w.Body.String() != "custom 404" {
		t.Errorf("expected 'custom 404', got %q", w.Body.String())
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/", nil},
		{"", nil},
		{"/hello", []string{"hello"}},
		{"/a/b/c", []string{"a", "b", "c"}},
		{"/a/b/c/", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		result := splitPath(tt.path)
		if len(result) != len(tt.expected) {
			t.Errorf("splitPath(%q): expected %v, got %v", tt.path, tt.expected, result)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("splitPath(%q)[%d]: expected %q, got %q", tt.path, i, tt.expected[i], result[i])
			}
		}
	}
}

func TestAdapter_ImplementsInterface(t *testing.T) {
	var _ platform.HTTPAdapter = (*Adapter)(nil)
}
