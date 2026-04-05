package gonest

import (
	"net/http/httptest"
	"testing"
)

func TestFileTypeValidator_Allowed(t *testing.T) {
	v := &FileTypeValidator{AllowedTypes: []string{".jpg", ".png"}}
	file := &UploadedFile{Filename: "photo.jpg"}

	err := v.Validate(file)
	if err != nil {
		t.Errorf("expected no error for allowed type, got %v", err)
	}
}

func TestFileTypeValidator_NotAllowed(t *testing.T) {
	v := &FileTypeValidator{AllowedTypes: []string{".jpg", ".png"}}
	file := &UploadedFile{Filename: "script.exe"}

	err := v.Validate(file)
	if err == nil {
		t.Fatal("expected error for disallowed type")
	}
}

func TestFileTypeValidator_Nil(t *testing.T) {
	v := &FileTypeValidator{AllowedTypes: []string{".jpg"}}
	err := v.Validate(nil)
	if err != nil {
		t.Errorf("expected no error for nil file, got %v", err)
	}
}

func TestFileSizeValidator_WithinLimit(t *testing.T) {
	v := &FileSizeValidator{MaxSize: 10 << 20} // 10MB
	file := &UploadedFile{Size: 5 << 20}       // 5MB

	err := v.Validate(file)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFileSizeValidator_ExceedsLimit(t *testing.T) {
	v := &FileSizeValidator{MaxSize: 1 << 20} // 1MB
	file := &UploadedFile{Size: 5 << 20}      // 5MB

	err := v.Validate(file)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
}

func TestFileSizeValidator_Nil(t *testing.T) {
	v := &FileSizeValidator{MaxSize: 1 << 20}
	err := v.Validate(nil)
	if err != nil {
		t.Errorf("expected no error for nil file, got %v", err)
	}
}

func TestGetUploadedFile_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	file := GetUploadedFile(ctx, "avatar")
	if file != nil {
		t.Error("expected nil for missing file")
	}
}

func TestUploadedFile_Fields(t *testing.T) {
	file := &UploadedFile{
		Filename: "photo.jpg",
		Size:     1024,
		MimeType: "image/jpeg",
	}
	if file.Filename != "photo.jpg" {
		t.Error("unexpected filename")
	}
	if file.Size != 1024 {
		t.Error("unexpected size")
	}
	if file.MimeType != "image/jpeg" {
		t.Error("unexpected mime type")
	}
}
