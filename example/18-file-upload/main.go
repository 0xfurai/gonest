package main

import (
	"log"
	"net/http"

	"github.com/gonest"
)

// --- Controller ---

type UploadController struct{}

func NewUploadController() *UploadController { return &UploadController{} }

func (c *UploadController) Register(r gonest.Router) {
	r.Get("/", c.home)

	// Basic file upload (no validation)
	r.Post("/upload", c.upload).
		Interceptors(gonest.NewFileInterceptor("file"))

	// Upload with type and size validation
	r.Post("/upload/validated", c.uploadValidated).
		Interceptors(gonest.NewFileInterceptor("file", 5<<20)) // 5MB max
}

func (c *UploadController) home(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "File upload example",
	})
}

func (c *UploadController) upload(ctx gonest.Context) error {
	file := gonest.GetUploadedFile(ctx, "file")
	if file == nil {
		return gonest.NewBadRequestException("no file uploaded")
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"filename": file.Filename,
		"size":     file.Size,
		"mimeType": file.MimeType,
	})
}

func (c *UploadController) uploadValidated(ctx gonest.Context) error {
	file := gonest.GetUploadedFile(ctx, "file")
	if file == nil {
		return gonest.NewBadRequestException("no file uploaded")
	}

	// Validate file type
	typeValidator := &gonest.FileTypeValidator{AllowedTypes: []string{".jpg", ".jpeg", ".png"}}
	if err := typeValidator.Validate(file); err != nil {
		return err
	}

	// Validate file size
	sizeValidator := &gonest.FileSizeValidator{MaxSize: 5 << 20} // 5MB
	if err := sizeValidator.Validate(file); err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"filename": file.Filename,
		"size":     file.Size,
		"mimeType": file.MimeType,
	})
}

// --- Module ---

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewUploadController},
})

func main() {
	app := gonest.Create(AppModule)
	app.EnableCors()
	log.Fatal(app.Listen(":3000"))
}
