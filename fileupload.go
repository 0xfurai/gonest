package gonest

import (
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

// UploadedFile wraps a multipart file with metadata.
type UploadedFile struct {
	File     multipart.File
	Header   *multipart.FileHeader
	Filename string
	Size     int64
	MimeType string
}

// FileInterceptor extracts a single file from the request and stores it
// in the context store under the given field name.
type FileInterceptor struct {
	fieldName string
	maxSize   int64
}

// NewFileInterceptor creates an interceptor that extracts a file upload.
func NewFileInterceptor(fieldName string, maxSize ...int64) *FileInterceptor {
	size := int64(10 << 20) // 10MB default
	if len(maxSize) > 0 {
		size = maxSize[0]
	}
	return &FileInterceptor{fieldName: fieldName, maxSize: size}
}

func (fi *FileInterceptor) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	r := ctx.Request()

	if err := r.ParseMultipartForm(fi.maxSize); err != nil {
		return nil, NewBadRequestException("failed to parse multipart form: " + err.Error())
	}

	file, header, err := r.FormFile(fi.fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return next.Handle()
		}
		return nil, NewBadRequestException("file upload error: " + err.Error())
	}

	uploaded := &UploadedFile{
		File:     file,
		Header:   header,
		Filename: header.Filename,
		Size:     header.Size,
		MimeType: header.Header.Get("Content-Type"),
	}

	ctx.Set("__file_"+fi.fieldName, uploaded)

	return next.Handle()
}

// GetUploadedFile retrieves an uploaded file from the context.
func GetUploadedFile(ctx Context, fieldName string) *UploadedFile {
	val, ok := ctx.Get("__file_" + fieldName)
	if !ok {
		return nil
	}
	return val.(*UploadedFile)
}

// FileTypeValidator validates that an uploaded file has an allowed extension.
type FileTypeValidator struct {
	AllowedTypes []string // e.g., ".jpg", ".png", ".pdf"
}

// Validate checks if the file extension is allowed.
func (v *FileTypeValidator) Validate(file *UploadedFile) error {
	if file == nil {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	for _, allowed := range v.AllowedTypes {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}
	return NewBadRequestException("file type " + ext + " is not allowed")
}

// FileSizeValidator validates that an uploaded file is within size limits.
type FileSizeValidator struct {
	MaxSize int64
}

// Validate checks if the file is within the size limit.
func (v *FileSizeValidator) Validate(file *UploadedFile) error {
	if file == nil {
		return nil
	}
	if file.Size > v.MaxSize {
		return NewPayloadTooLargeException("file too large")
	}
	return nil
}

// ReadFileBytes reads all bytes from an uploaded file.
func ReadFileBytes(file *UploadedFile) ([]byte, error) {
	if file == nil {
		return nil, NewBadRequestException("no file uploaded")
	}
	defer file.File.Close()
	return io.ReadAll(file.File)
}

// FilesInterceptor extracts multiple files from a single form field.
// Equivalent to NestJS FilesInterceptor.
type FilesInterceptor struct {
	fieldName string
	maxCount  int
	maxSize   int64
}

// NewFilesInterceptor creates an interceptor that extracts multiple files
// from a single form field. maxCount limits the number of files (0 = unlimited).
func NewFilesInterceptor(fieldName string, maxCount int, maxSize ...int64) *FilesInterceptor {
	size := int64(10 << 20) // 10MB default
	if len(maxSize) > 0 {
		size = maxSize[0]
	}
	return &FilesInterceptor{fieldName: fieldName, maxCount: maxCount, maxSize: size}
}

func (fi *FilesInterceptor) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	r := ctx.Request()
	if err := r.ParseMultipartForm(fi.maxSize); err != nil {
		return nil, NewBadRequestException("failed to parse multipart form: " + err.Error())
	}

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return next.Handle()
	}

	headers := r.MultipartForm.File[fi.fieldName]
	if fi.maxCount > 0 && len(headers) > fi.maxCount {
		return nil, NewBadRequestException("too many files uploaded")
	}

	var files []*UploadedFile
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return nil, NewBadRequestException("file upload error: " + err.Error())
		}
		files = append(files, &UploadedFile{
			File:     file,
			Header:   header,
			Filename: header.Filename,
			Size:     header.Size,
			MimeType: header.Header.Get("Content-Type"),
		})
	}

	ctx.Set("__files_"+fi.fieldName, files)
	return next.Handle()
}

// GetUploadedFiles retrieves multiple uploaded files from the context.
func GetUploadedFiles(ctx Context, fieldName string) []*UploadedFile {
	val, ok := ctx.Get("__files_" + fieldName)
	if !ok {
		return nil
	}
	return val.([]*UploadedFile)
}

// FileFieldSpec describes a named file field for FileFieldsInterceptor.
type FileFieldSpec struct {
	Name     string
	MaxCount int
}

// FileFieldsInterceptor extracts files from multiple named form fields.
// Equivalent to NestJS FileFieldsInterceptor.
type FileFieldsInterceptor struct {
	fields  []FileFieldSpec
	maxSize int64
}

// NewFileFieldsInterceptor creates an interceptor that extracts files from
// multiple named fields.
func NewFileFieldsInterceptor(fields []FileFieldSpec, maxSize ...int64) *FileFieldsInterceptor {
	size := int64(10 << 20) // 10MB default
	if len(maxSize) > 0 {
		size = maxSize[0]
	}
	return &FileFieldsInterceptor{fields: fields, maxSize: size}
}

func (fi *FileFieldsInterceptor) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	r := ctx.Request()
	if err := r.ParseMultipartForm(fi.maxSize); err != nil {
		return nil, NewBadRequestException("failed to parse multipart form: " + err.Error())
	}

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return next.Handle()
	}

	for _, field := range fi.fields {
		headers := r.MultipartForm.File[field.Name]
		if field.MaxCount > 0 && len(headers) > field.MaxCount {
			return nil, NewBadRequestException("too many files for field " + field.Name)
		}

		var files []*UploadedFile
		for _, header := range headers {
			file, err := header.Open()
			if err != nil {
				return nil, NewBadRequestException("file upload error: " + err.Error())
			}
			files = append(files, &UploadedFile{
				File:     file,
				Header:   header,
				Filename: header.Filename,
				Size:     header.Size,
				MimeType: header.Header.Get("Content-Type"),
			})
		}
		ctx.Set("__files_"+field.Name, files)
	}

	return next.Handle()
}

// AnyFilesInterceptor extracts all uploaded files regardless of field name.
// Equivalent to NestJS AnyFilesInterceptor.
type AnyFilesInterceptor struct {
	maxSize int64
}

// NewAnyFilesInterceptor creates an interceptor that extracts all uploaded files.
func NewAnyFilesInterceptor(maxSize ...int64) *AnyFilesInterceptor {
	size := int64(10 << 20) // 10MB default
	if len(maxSize) > 0 {
		size = maxSize[0]
	}
	return &AnyFilesInterceptor{maxSize: size}
}

func (fi *AnyFilesInterceptor) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	r := ctx.Request()
	if err := r.ParseMultipartForm(fi.maxSize); err != nil {
		return nil, NewBadRequestException("failed to parse multipart form: " + err.Error())
	}

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return next.Handle()
	}

	var allFiles []*UploadedFile
	for fieldName, headers := range r.MultipartForm.File {
		for _, header := range headers {
			file, err := header.Open()
			if err != nil {
				return nil, NewBadRequestException("file upload error: " + err.Error())
			}
			uf := &UploadedFile{
				File:     file,
				Header:   header,
				Filename: header.Filename,
				Size:     header.Size,
				MimeType: header.Header.Get("Content-Type"),
			}
			allFiles = append(allFiles, uf)
			// Also store per-field
			ctx.Set("__file_"+fieldName, uf)
		}
	}

	ctx.Set("__files_all", allFiles)
	return next.Handle()
}

// GetAllUploadedFiles retrieves all uploaded files from the context.
func GetAllUploadedFiles(ctx Context) []*UploadedFile {
	val, ok := ctx.Get("__files_all")
	if !ok {
		return nil
	}
	return val.([]*UploadedFile)
}

// NoFilesInterceptor rejects requests that contain file uploads.
// Equivalent to NestJS NoFilesInterceptor.
type NoFilesInterceptor struct{}

// NewNoFilesInterceptor creates an interceptor that rejects file uploads.
func NewNoFilesInterceptor() *NoFilesInterceptor {
	return &NoFilesInterceptor{}
}

func (fi *NoFilesInterceptor) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	r := ctx.Request()
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(1 << 20); err == nil {
			if r.MultipartForm != nil && len(r.MultipartForm.File) > 0 {
				return nil, NewBadRequestException("file upload is not allowed")
			}
		}
	}
	return next.Handle()
}
