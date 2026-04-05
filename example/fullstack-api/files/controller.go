package files

import (
	"net/http"

	"github.com/gonest"
	"github.com/gonest/example/fullstack-api/common"
)

// FilesController handles /files endpoints.
type FilesController struct {
	service *FilesService
}

func NewFilesController(service *FilesService) *FilesController {
	return &FilesController{service: service}
}

func (c *FilesController) Register(r gonest.Router) {
	r.Prefix("/api/v1/files")

	// Upload a file (max 5MB, authenticated)
	r.Post("/upload", c.upload).
		Summary("Upload a file").
		Response(201, FileRecord{}).
		Interceptors(gonest.NewFileInterceptor("file", 5<<20))

	// List all uploaded files
	r.Get("/", c.findAll).
		Summary("List all files").
		Response(200, []FileRecord{})

	// Get file info by ID
	r.Get("/:id", c.findOne).
		Summary("Get file info by ID").
		Pipes(gonest.NewParseIntPipe("id")).
		Response(200, FileRecord{})

	// Delete a file (admin only)
	r.Delete("/:id", c.remove).
		Summary("Delete a file (admin)").
		Pipes(gonest.NewParseIntPipe("id")).
		Response(204, nil).
		SetMetadata("roles", []common.Role{common.RoleAdmin}).
		Guards(&common.RolesGuard{})
}

func (c *FilesController) upload(ctx gonest.Context) error {
	user, ok := ctx.Get("user")
	if !ok {
		return gonest.NewUnauthorizedException("not authenticated")
	}

	file := gonest.GetUploadedFile(ctx, "file")
	if file == nil {
		return gonest.NewBadRequestException("file is required")
	}

	record, err := c.service.Upload(user.(*common.AuthUser).ID, file)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, record)
}

func (c *FilesController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.service.FindAll())
}

func (c *FilesController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	record := c.service.FindByID(id)
	if record == nil {
		return gonest.NewNotFoundException("file not found")
	}
	return ctx.JSON(http.StatusOK, record)
}

func (c *FilesController) remove(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	if err := c.service.Delete(id); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}
