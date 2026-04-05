package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonest"
)

func createTestApp(t *testing.T) *gonest.Application {
	t.Helper()
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestFindAllExcludesPassword(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/users/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var users []map[string]any
	json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) == 0 {
		t.Fatal("expected users in response")
	}

	for _, u := range users {
		if _, hasPassword := u["password"]; hasPassword {
			t.Errorf("expected password to be excluded, found in user %v", u["name"])
		}
		// Name and email should still be present
		if u["name"] == nil {
			t.Error("expected name field")
		}
		if u["email"] == nil {
			t.Error("expected email field")
		}
	}
}

func TestFindAllExcludesSSNForNonAdmin(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/users/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	var users []map[string]any
	json.Unmarshal(w.Body.Bytes(), &users)

	for _, u := range users {
		if _, hasSSN := u["ssn"]; hasSSN {
			t.Errorf("expected SSN to be hidden for non-admin, found in user %v", u["name"])
		}
	}
}

func TestFindAllAdminIncludesSSN(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/users/admin", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var users []map[string]any
	json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) == 0 {
		t.Fatal("expected users in response")
	}

	hasSSN := false
	for _, u := range users {
		if u["ssn"] != nil && u["ssn"] != "" {
			hasSSN = true
			break
		}
	}
	if !hasSSN {
		t.Error("expected SSN to be visible for admin group")
	}

	// Password should still be excluded even for admin
	for _, u := range users {
		if _, hasPassword := u["password"]; hasPassword {
			t.Error("expected password to still be excluded for admin")
		}
	}
}
