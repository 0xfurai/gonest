package gonest

import (
	"testing"
)

type validationTestDto struct {
	Name  string `validate:"required,min=2,max=50"`
	Age   int    `validate:"required,gte=0,lte=200"`
	Email string `validate:"required,email"`
}

func TestValidationPipe_Valid(t *testing.T) {
	pipe := NewValidationPipe()
	dto := validationTestDto{
		Name:  "John",
		Age:   25,
		Email: "john@example.com",
	}

	result, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(validationTestDto).Name != "John" {
		t.Error("expected passthrough")
	}
}

func TestValidationPipe_RequiredFieldMissing(t *testing.T) {
	pipe := NewValidationPipe()
	dto := validationTestDto{
		Name: "",
		Age:  25,
	}

	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	httpErr := err.(*HTTPException)
	if httpErr.StatusCode() != 400 {
		t.Errorf("expected 400, got %d", httpErr.StatusCode())
	}
}

func TestValidationPipe_MinLength(t *testing.T) {
	pipe := NewValidationPipe()
	dto := validationTestDto{
		Name:  "J", // min=2
		Age:   25,
		Email: "j@x.co",
	}

	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected validation error for min length")
	}
}

func TestValidationPipe_GteViolation(t *testing.T) {
	pipe := NewValidationPipe()
	dto := validationTestDto{
		Name:  "John",
		Age:   -1, // gte=0
		Email: "john@example.com",
	}

	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected validation error for gte")
	}
}

func TestValidationPipe_InvalidEmail(t *testing.T) {
	pipe := NewValidationPipe()
	dto := validationTestDto{
		Name:  "John",
		Age:   25,
		Email: "not-an-email",
	}

	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected validation error for email")
	}
}

func TestValidationPipe_SkipNonBody(t *testing.T) {
	pipe := NewValidationPipe()
	result, err := pipe.Transform("any value", ArgumentMetadata{Type: "param"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "any value" {
		t.Error("expected passthrough for non-body metadata")
	}
}

func TestValidationPipe_NilBody(t *testing.T) {
	pipe := NewValidationPipe()
	_, err := pipe.Transform(nil, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected error for nil body")
	}
}

func TestValidationPipe_Pointer(t *testing.T) {
	pipe := NewValidationPipe()
	dto := &validationTestDto{
		Name:  "John",
		Age:   25,
		Email: "john@example.com",
	}

	result, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestValidationPipe_MaxLength(t *testing.T) {
	pipe := NewValidationPipe()

	type shortName struct {
		Name string `validate:"max=3"`
	}

	dto := shortName{Name: "Jonathan"}
	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected validation error for max length")
	}
}

func TestValidationPipe_LteViolation(t *testing.T) {
	pipe := NewValidationPipe()

	type ageLimit struct {
		Age int `validate:"lte=100"`
	}

	dto := ageLimit{Age: 150}
	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected validation error for lte")
	}
}

func TestValidationPipe_Omitempty_SkipsWhenEmpty(t *testing.T) {
	pipe := NewValidationPipe()

	type updateDto struct {
		Name   string `validate:"omitempty,min=3,max=50"`
		Status string `validate:"omitempty,oneof=active inactive"`
	}

	// All zero values — should pass because of omitempty.
	dto := updateDto{}
	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err != nil {
		t.Fatalf("expected no error for empty omitempty fields, got: %v", err)
	}
}

func TestValidationPipe_Omitempty_ValidatesWhenPresent(t *testing.T) {
	pipe := NewValidationPipe()

	type updateDto struct {
		Name string `validate:"omitempty,min=3"`
	}

	dto := updateDto{Name: "ab"} // provided but too short
	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected validation error for min length when value is present")
	}
}

func TestValidationPipe_Oneof_Valid(t *testing.T) {
	pipe := NewValidationPipe()

	type statusDto struct {
		Status string `validate:"required,oneof=draft published"`
	}

	dto := statusDto{Status: "draft"}
	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidationPipe_Oneof_Invalid(t *testing.T) {
	pipe := NewValidationPipe()

	type statusDto struct {
		Status string `validate:"required,oneof=draft published"`
	}

	dto := statusDto{Status: "deleted"}
	_, err := pipe.Transform(dto, ArgumentMetadata{Type: "body"})
	if err == nil {
		t.Fatal("expected validation error for invalid oneof value")
	}
}
