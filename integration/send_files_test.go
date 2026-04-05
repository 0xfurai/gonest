package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xfurai/gonest"
)

// ---------------------------------------------------------------------------
// Send Files Integration Tests
// Mirror: original/integration/send-files/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// File controller
// ---------------------------------------------------------------------------

type fileController struct {
	tempDir string
}

func newFileController(tempDir string) *fileController {
	return &fileController{tempDir: tempDir}
}

func (c *fileController) Register(r gonest.Router) {
	r.Prefix("/file")
	r.Get("/stream", c.streamFile)
	r.Get("/buffer", c.bufferFile)
	r.Get("/not-found", c.notFoundFile)
	r.Get("/custom-headers", c.customHeaders)
}

func (c *fileController) streamFile(ctx gonest.Context) error {
	path := filepath.Join(c.tempDir, "test.txt")
	f, err := os.Open(path)
	if err != nil {
		return gonest.NewNotFoundException("file not found")
	}
	defer f.Close()

	stat, _ := f.Stat()
	ctx.SetHeader("Content-Type", "text/plain")
	ctx.SetHeader("Content-Length", fmt.Sprintf("%d", stat.Size()))
	ctx.ResponseWriter().WriteHeader(http.StatusOK)
	io.Copy(ctx.ResponseWriter(), f)
	return nil
}

func (c *fileController) bufferFile(ctx gonest.Context) error {
	path := filepath.Join(c.tempDir, "test.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return gonest.NewNotFoundException("file not found")
	}

	ctx.SetHeader("Content-Type", "application/octet-stream")
	ctx.SetHeader("Content-Disposition", "attachment; filename=test.txt")
	ctx.ResponseWriter().WriteHeader(http.StatusOK)
	ctx.ResponseWriter().Write(data)
	return nil
}

func (c *fileController) notFoundFile(ctx gonest.Context) error {
	return gonest.NewNotFoundException("file not found")
}

func (c *fileController) customHeaders(ctx gonest.Context) error {
	path := filepath.Join(c.tempDir, "test.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return gonest.NewNotFoundException("file not found")
	}

	ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
	ctx.SetHeader("Content-Disposition", "inline; filename=readme.txt")
	ctx.SetHeader("X-Custom-File-Header", "present")
	ctx.ResponseWriter().WriteHeader(http.StatusOK)
	ctx.ResponseWriter().Write(data)
	return nil
}

func createFileApp(t *testing.T) (*gonest.Application, string) {
	t.Helper()

	// Create temp directory with test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello, this is a test file!"), 0644)

	ctrl := newFileController(tempDir)
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *fileController { return ctrl }},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	return app, tempDir
}

// ---------------------------------------------------------------------------
// Tests: Stream file
// ---------------------------------------------------------------------------

func TestSendFiles_StreamFile(t *testing.T) {
	app, _ := createFileApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/file/stream", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("expected text/plain, got %q", w.Header().Get("Content-Type"))
	}
	if w.Body.String() != "Hello, this is a test file!" {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Tests: Buffer file
// ---------------------------------------------------------------------------

func TestSendFiles_BufferFile(t *testing.T) {
	app, _ := createFileApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/file/buffer", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("expected application/octet-stream, got %q", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Content-Disposition") != "attachment; filename=test.txt" {
		t.Errorf("expected attachment disposition, got %q", w.Header().Get("Content-Disposition"))
	}
	if !bytes.Equal(w.Body.Bytes(), []byte("Hello, this is a test file!")) {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Tests: File not found
// ---------------------------------------------------------------------------

func TestSendFiles_NotFound(t *testing.T) {
	app, _ := createFileApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/file/not-found", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "file not found" {
		t.Errorf("expected 'file not found', got %v", body["message"])
	}
}

// ---------------------------------------------------------------------------
// Tests: Custom headers
// ---------------------------------------------------------------------------

func TestSendFiles_CustomHeaders(t *testing.T) {
	app, _ := createFileApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/file/custom-headers", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Disposition") != "inline; filename=readme.txt" {
		t.Errorf("expected inline disposition, got %q", w.Header().Get("Content-Disposition"))
	}
	if w.Header().Get("X-Custom-File-Header") != "present" {
		t.Errorf("expected X-Custom-File-Header")
	}
}

// ---------------------------------------------------------------------------
// Tests: Large file streaming
// ---------------------------------------------------------------------------

func TestSendFiles_LargeFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a 1MB file
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	os.WriteFile(filepath.Join(tempDir, "test.txt"), largeData, 0644)

	ctrl := newFileController(tempDir)
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *fileController { return ctrl }},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/file/stream", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.Len() != len(largeData) {
		t.Errorf("expected %d bytes, got %d", len(largeData), w.Body.Len())
	}
}
