package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xfurai/gonest"
)

func createTestApp(t *testing.T) *gonest.Application {
	t.Helper()
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestGetCatsEmpty(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/cats/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var cats []Cat
	json.Unmarshal(w.Body.Bytes(), &cats)
	if len(cats) != 0 {
		t.Errorf("expected empty list, got %d cats", len(cats))
	}
}

func TestCreateCat(t *testing.T) {
	app := createTestApp(t)

	body := `{"name":"Pixel","age":3,"breed":"Bombay"}`
	req := httptest.NewRequest("POST", "/cats/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var cat Cat
	json.Unmarshal(w.Body.Bytes(), &cat)
	if cat.Name != "Pixel" {
		t.Errorf("expected name 'Pixel', got %q", cat.Name)
	}
	if cat.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestGetCatById(t *testing.T) {
	app := createTestApp(t)

	// Create a cat first
	body := `{"name":"Luna","age":2,"breed":"Persian"}`
	req := httptest.NewRequest("POST", "/cats/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	var created Cat
	json.Unmarshal(w.Body.Bytes(), &created)

	// Get by ID
	req = httptest.NewRequest("GET", "/cats/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var cat Cat
	json.Unmarshal(w.Body.Bytes(), &cat)
	if cat.Name != "Luna" {
		t.Errorf("expected 'Luna', got %q", cat.Name)
	}
}

func TestGetCatNotFound(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/cats/999", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteCat(t *testing.T) {
	app := createTestApp(t)

	// Create a cat
	body := `{"name":"Milo","age":1,"breed":"Siamese"}`
	req := httptest.NewRequest("POST", "/cats/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	var created Cat
	json.Unmarshal(w.Body.Bytes(), &created)

	// Delete the cat
	req = httptest.NewRequest("DELETE", "/cats/"+itoa(created.ID), nil)
	req.Header.Set("X-User-Role", "admin")
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	// Verify it's gone
	req = httptest.NewRequest("GET", "/cats/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestCreateCatForbiddenNoRole(t *testing.T) {
	app := createTestApp(t)

	body := `{"name":"Nope","age":1,"breed":"Tabby"}`
	req := httptest.NewRequest("POST", "/cats/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-User-Role header
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCreateCatForbiddenWrongRole(t *testing.T) {
	app := createTestApp(t)

	body := `{"name":"Nope","age":1,"breed":"Tabby"}`
	req := httptest.NewRequest("POST", "/cats/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "user")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestInvalidCatId(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/cats/abc", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
