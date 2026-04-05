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

func TestLoginSuccess(t *testing.T) {
	app := createTestApp(t)

	body := `{"username":"admin","password":"password"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] == "" {
		t.Error("expected access_token in response")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	app := createTestApp(t)

	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestProfileWithToken(t *testing.T) {
	app := createTestApp(t)

	// Login first
	body := `{"username":"admin","password":"password"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	var loginResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp["access_token"]

	// Access profile with token
	req = httptest.NewRequest("GET", "/profile/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var profile map[string]string
	json.Unmarshal(w.Body.Bytes(), &profile)
	if profile["id"] != "1" {
		t.Errorf("expected id '1', got %q", profile["id"])
	}
	if profile["role"] != "admin" {
		t.Errorf("expected role 'admin', got %q", profile["role"])
	}
}

func TestProfileWithoutToken(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/profile/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestProfileWithInvalidToken(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/profile/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
