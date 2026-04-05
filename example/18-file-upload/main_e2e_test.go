package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

func createMultipartFile(t *testing.T, fieldName, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	writer.Close()
	return &buf, writer.FormDataContentType()
}

func TestHome(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "File upload example" {
		t.Errorf("expected home message, got %q", body["message"])
	}
}

func TestUploadFile(t *testing.T) {
	app := createTestApp(t)

	buf, contentType := createMultipartFile(t, "file", "test.txt", []byte("hello world"))
	req := httptest.NewRequest("POST", "/upload", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["filename"] != "test.txt" {
		t.Errorf("expected filename 'test.txt', got %v", resp["filename"])
	}
	if resp["size"] != float64(11) {
		t.Errorf("expected size 11, got %v", resp["size"])
	}
}

func TestUploadNoFile(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("POST", "/upload", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUploadValidatedSuccess(t *testing.T) {
	app := createTestApp(t)

	buf, contentType := createMultipartFile(t, "file", "photo.jpg", []byte("fake jpg data"))
	req := httptest.NewRequest("POST", "/upload/validated", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["filename"] != "photo.jpg" {
		t.Errorf("expected filename 'photo.jpg', got %v", resp["filename"])
	}
}

func TestUploadValidatedWrongType(t *testing.T) {
	app := createTestApp(t)

	buf, contentType := createMultipartFile(t, "file", "virus.exe", []byte("bad data"))
	req := httptest.NewRequest("POST", "/upload/validated", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", w.Code, w.Body.String())
	}
}
