package gonest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultExceptionFilter_HTTPException(t *testing.T) {
	filter := &DefaultExceptionFilter{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	ctx := newContext(w, r)
	host := newArgumentsHost(ctx)

	err := NewBadRequestException("invalid input")
	filter.Catch(err, host)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["statusCode"] != float64(400) {
		t.Errorf("expected statusCode 400, got %v", body["statusCode"])
	}
	if body["message"] != "invalid input" {
		t.Errorf("expected 'invalid input', got %v", body["message"])
	}
	if body["path"] != "/test" {
		t.Errorf("expected '/test', got %v", body["path"])
	}
	if body["timestamp"] == nil {
		t.Error("expected timestamp")
	}
}

func TestDefaultExceptionFilter_GenericError(t *testing.T) {
	filter := &DefaultExceptionFilter{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	host := newArgumentsHost(ctx)

	filter.Catch(http.ErrAbortHandler, host)

	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestExceptionFilterFunc(t *testing.T) {
	called := false
	filter := ExceptionFilterFunc(func(err error, host ArgumentsHost) error {
		called = true
		httpCtx := host.SwitchToHTTP()
		resp := httpCtx.Response()
		resp.WriteHeader(422)
		resp.Write([]byte("custom"))
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	host := newArgumentsHost(ctx)

	filter.Catch(NewBadRequestException("test"), host)
	if !called {
		t.Error("expected filter to be called")
	}
	if w.Code != 422 {
		t.Errorf("expected 422, got %d", w.Code)
	}
}
