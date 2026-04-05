package gonest

import (
	"testing"
	"time"
)

func TestParseIntPipe_ValidInt(t *testing.T) {
	pipe := NewParseIntPipe("id")
	meta := ArgumentMetadata{Type: "param", Name: "id"}

	result, err := pipe.Transform("42", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestParseIntPipe_InvalidInt(t *testing.T) {
	pipe := NewParseIntPipe("id")
	meta := ArgumentMetadata{Type: "param", Name: "id"}

	_, err := pipe.Transform("abc", meta)
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*HTTPException)
	if !ok {
		t.Fatal("expected HTTPException")
	}
	if httpErr.StatusCode() != 400 {
		t.Errorf("expected 400, got %d", httpErr.StatusCode())
	}
}

func TestParseIntPipe_DifferentParam(t *testing.T) {
	pipe := NewParseIntPipe("id")
	meta := ArgumentMetadata{Type: "param", Name: "other"}

	result, err := pipe.Transform("abc", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "abc" {
		t.Errorf("expected 'abc' passthrough, got %v", result)
	}
}

func TestParseBoolPipe_Valid(t *testing.T) {
	pipe := NewParseBoolPipe("active")
	meta := ArgumentMetadata{Type: "param", Name: "active"}

	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
	}

	for _, tt := range tests {
		result, err := pipe.Transform(tt.input, meta)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("for %q: expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestParseBoolPipe_Invalid(t *testing.T) {
	pipe := NewParseBoolPipe("active")
	meta := ArgumentMetadata{Type: "param", Name: "active"}

	_, err := pipe.Transform("maybe", meta)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFloatPipe_Valid(t *testing.T) {
	pipe := NewParseFloatPipe("price")
	meta := ArgumentMetadata{Type: "param", Name: "price"}

	result, err := pipe.Transform("3.14", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 3.14 {
		t.Errorf("expected 3.14, got %v", result)
	}
}

func TestParseFloatPipe_Invalid(t *testing.T) {
	pipe := NewParseFloatPipe("price")
	meta := ArgumentMetadata{Type: "param", Name: "price"}

	_, err := pipe.Transform("not-a-number", meta)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseUUIDPipe_Valid(t *testing.T) {
	pipe := NewParseUUIDPipe("id")
	meta := ArgumentMetadata{Type: "param", Name: "id"}

	result, err := pipe.Transform("550e8400-e29b-41d4-a716-446655440000", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestParseUUIDPipe_Invalid(t *testing.T) {
	pipe := NewParseUUIDPipe("id")
	meta := ArgumentMetadata{Type: "param", Name: "id"}

	_, err := pipe.Transform("not-a-uuid", meta)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultValuePipe_NilValue(t *testing.T) {
	pipe := NewDefaultValuePipe("page", 1)
	meta := ArgumentMetadata{Type: "query", Name: "page"}

	result, err := pipe.Transform(nil, meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestDefaultValuePipe_EmptyString(t *testing.T) {
	pipe := NewDefaultValuePipe("page", 1)
	meta := ArgumentMetadata{Type: "query", Name: "page"}

	result, err := pipe.Transform("", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestDefaultValuePipe_HasValue(t *testing.T) {
	pipe := NewDefaultValuePipe("page", 1)
	meta := ArgumentMetadata{Type: "query", Name: "page"}

	result, err := pipe.Transform("5", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "5" {
		t.Errorf("expected '5', got %v", result)
	}
}

func TestParseArrayPipe_Valid(t *testing.T) {
	pipe := NewParseArrayPipe("tags")
	meta := ArgumentMetadata{Type: "query", Name: "tags"}

	result, err := pipe.Transform("a, b, c", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := result.([]string)
	if len(arr) != 3 || arr[0] != "a" || arr[1] != "b" || arr[2] != "c" {
		t.Errorf("expected [a b c], got %v", arr)
	}
}

func TestParseArrayPipe_Empty(t *testing.T) {
	pipe := NewParseArrayPipe("tags")
	meta := ArgumentMetadata{Type: "query", Name: "tags"}

	result, err := pipe.Transform("", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := result.([]string)
	if len(arr) != 0 {
		t.Errorf("expected empty array, got %v", arr)
	}
}

func TestPipeFunc(t *testing.T) {
	pipe := PipeFunc(func(value any, meta ArgumentMetadata) (any, error) {
		return "transformed", nil
	})

	result, err := pipe.Transform("input", ArgumentMetadata{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "transformed" {
		t.Errorf("expected 'transformed', got %v", result)
	}
}

func TestParseIntPipe_NonStringValue(t *testing.T) {
	pipe := NewParseIntPipe("id")
	meta := ArgumentMetadata{Type: "param", Name: "id"}

	_, err := pipe.Transform(123, meta)
	if err == nil {
		t.Fatal("expected error for non-string input")
	}
}

func TestParseDatePipe_RFC3339(t *testing.T) {
	pipe := NewParseDatePipe("date")
	meta := ArgumentMetadata{Type: "param", Name: "date"}

	result, err := pipe.Transform("2024-01-15T10:30:00Z", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.(time.Time); !ok {
		t.Errorf("expected time.Time, got %T", result)
	}
}

func TestParseDatePipe_DateOnly(t *testing.T) {
	pipe := NewParseDatePipe("date")
	meta := ArgumentMetadata{Type: "param", Name: "date"}

	result, err := pipe.Transform("2024-01-15", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.(time.Time); !ok {
		t.Errorf("expected time.Time, got %T", result)
	}
}

func TestParseDatePipe_Invalid(t *testing.T) {
	pipe := NewParseDatePipe("date")
	meta := ArgumentMetadata{Type: "param", Name: "date"}

	_, err := pipe.Transform("not-a-date", meta)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestParseDatePipe_CustomFormat(t *testing.T) {
	pipe := &ParseDatePipe{ParamName: "date", Format: "02/01/2006"}
	meta := ArgumentMetadata{Type: "param", Name: "date"}

	result, err := pipe.Transform("15/01/2024", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.(time.Time); !ok {
		t.Errorf("expected time.Time, got %T", result)
	}
}

func TestParseEnumPipe_Valid(t *testing.T) {
	pipe := NewParseEnumPipe("status", "active", "inactive", "pending")
	meta := ArgumentMetadata{Type: "param", Name: "status"}

	result, err := pipe.Transform("active", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "active" {
		t.Errorf("expected 'active', got %v", result)
	}
}

func TestParseEnumPipe_Invalid(t *testing.T) {
	pipe := NewParseEnumPipe("status", "active", "inactive", "pending")
	meta := ArgumentMetadata{Type: "param", Name: "status"}

	_, err := pipe.Transform("unknown", meta)
	if err == nil {
		t.Fatal("expected error for invalid enum value")
	}
}

func TestParseEnumPipe_SkipOtherParam(t *testing.T) {
	pipe := NewParseEnumPipe("status", "active", "inactive")
	meta := ArgumentMetadata{Type: "param", Name: "other"}

	result, err := pipe.Transform("anything", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "anything" {
		t.Errorf("expected passthrough, got %v", result)
	}
}

func TestParseFilePipeBuilder(t *testing.T) {
	pipe := NewParseFilePipeBuilder("avatar").
		AddFileTypeValidator(".jpg", ".png").
		AddMaxSizeValidator(1024 * 1024).
		Build()

	if pipe.FieldName != "avatar" {
		t.Errorf("expected field name 'avatar', got %q", pipe.FieldName)
	}
	if len(pipe.Validators) != 2 {
		t.Errorf("expected 2 validators, got %d", len(pipe.Validators))
	}
}
