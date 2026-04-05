package gonest

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestTemplateEngine_Render(t *testing.T) {
	// Create temp directory and template file
	dir := t.TempDir()
	tmplContent := `<h1>Hello {{.Name}}!</h1>`
	err := os.WriteFile(filepath.Join(dir, "greeting.html"), []byte(tmplContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	engine := NewTemplateEngine(dir)
	w := httptest.NewRecorder()

	err = engine.Render(w, "greeting", map[string]string{"Name": "World"})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	body := w.Body.String()
	if body != "<h1>Hello World!</h1>" {
		t.Errorf("expected '<h1>Hello World!</h1>', got %q", body)
	}
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("unexpected content type: %q", w.Header().Get("Content-Type"))
	}
}

func TestTemplateEngine_RenderCached(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "page.html"), []byte(`{{.Title}}`), 0644)

	engine := NewTemplateEngine(dir)

	w1 := httptest.NewRecorder()
	engine.Render(w1, "page", map[string]string{"Title": "First"})

	w2 := httptest.NewRecorder()
	engine.Render(w2, "page", map[string]string{"Title": "Second"})

	if w2.Body.String() != "Second" {
		t.Errorf("expected 'Second', got %q", w2.Body.String())
	}
}

func TestTemplateEngine_MissingTemplate(t *testing.T) {
	engine := NewTemplateEngine("/nonexistent/dir")
	w := httptest.NewRecorder()
	err := engine.Render(w, "missing", nil)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestTemplateEngine_SetExtension(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "page.hbs"), []byte(`HBS: {{.X}}`), 0644)

	engine := NewTemplateEngine(dir)
	engine.SetExtension(".hbs")

	w := httptest.NewRecorder()
	err := engine.Render(w, "page", map[string]string{"X": "works"})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if w.Body.String() != "HBS: works" {
		t.Errorf("expected 'HBS: works', got %q", w.Body.String())
	}
}

func TestTemplateEngine_AddFunc(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "func.html"), []byte(`{{upper .Name}}`), 0644)

	engine := NewTemplateEngine(dir)
	engine.AddFunc("upper", func(s string) string {
		result := make([]byte, len(s))
		for i, c := range s {
			if c >= 'a' && c <= 'z' {
				result[i] = byte(c - 32)
			} else {
				result[i] = byte(c)
			}
		}
		return string(result)
	})

	w := httptest.NewRecorder()
	err := engine.Render(w, "func", map[string]string{"Name": "hello"})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if w.Body.String() != "HELLO" {
		t.Errorf("expected 'HELLO', got %q", w.Body.String())
	}
}

func TestStaticFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{}"), 0644)

	handler := StaticFiles("/static/", dir)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/static/style.css", nil)
	ctx := newContext(w, r)

	handler(ctx)
	if w.Body.String() != "body{}" {
		t.Errorf("expected 'body{}', got %q", w.Body.String())
	}
}

func TestRenderHandler(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(`Welcome {{.User}}`), 0644)

	engine := NewTemplateEngine(dir)
	handler := RenderHandler(engine, "index", func(ctx Context) any {
		return map[string]string{"User": "Alice"}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	handler(ctx)

	if w.Body.String() != "Welcome Alice" {
		t.Errorf("expected 'Welcome Alice', got %q", w.Body.String())
	}
}
