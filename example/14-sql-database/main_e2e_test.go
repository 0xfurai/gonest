package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gonest"
	gosql "github.com/gonest/database/sql"

	_ "modernc.org/sqlite"
)

func createTestApp(t *testing.T) *gonest.Application {
	t.Helper()

	// Use in-memory SQLite for tests
	dbModule := gosql.NewModule(gosql.Options{
		Driver: gosql.DriverSQLite,
		// Empty Database = :memory:
	})

	usersModule := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{NewUsersController},
		Providers:   []any{NewUsersService},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{dbModule, usersModule},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestCreateUser(t *testing.T) {
	app := createTestApp(t)

	body := `{"firstName":"John","lastName":"Doe"}`
	req := httptest.NewRequest("POST", "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var user User
	json.Unmarshal(w.Body.Bytes(), &user)
	if user.FirstName != "John" {
		t.Errorf("expected firstName 'John', got %q", user.FirstName)
	}
	if user.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if !user.IsActive {
		t.Error("expected isActive to be true")
	}
}

func TestListUsers(t *testing.T) {
	app := createTestApp(t)

	// Create a user first
	body := `{"firstName":"Jane","lastName":"Smith"}`
	req := httptest.NewRequest("POST", "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	// List users
	req = httptest.NewRequest("GET", "/users/", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var users []User
	json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestGetUser(t *testing.T) {
	app := createTestApp(t)

	// Create
	body := `{"firstName":"Alice","lastName":"Wonder"}`
	req := httptest.NewRequest("POST", "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	var created User
	json.Unmarshal(w.Body.Bytes(), &created)

	// Get
	req = httptest.NewRequest("GET", "/users/1", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var user User
	json.Unmarshal(w.Body.Bytes(), &user)
	if user.FirstName != "Alice" {
		t.Errorf("expected 'Alice', got %q", user.FirstName)
	}
}

func TestGetUserNotFound(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/users/999", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteUser(t *testing.T) {
	app := createTestApp(t)

	// Create
	body := `{"firstName":"Bob","lastName":"Delete"}`
	req := httptest.NewRequest("POST", "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	// Delete
	req = httptest.NewRequest("DELETE", "/users/1", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	// Verify gone
	req = httptest.NewRequest("GET", "/users/1", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("DELETE", "/users/999", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestInvalidUserId(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/users/abc", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestEmptyUserList(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/users/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var users []User
	json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}
