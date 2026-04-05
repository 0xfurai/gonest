package users

import (
	"fmt"
	"net/http"

	"github.com/gonest"
	"github.com/gonest/example/fullstack-api/common"
)

// UsersController handles /users endpoints.
type UsersController struct {
	service *UsersService
}

func NewUsersController(service *UsersService) *UsersController {
	return &UsersController{service: service}
}

func (c *UsersController) Register(r gonest.Router) {
	r.Prefix("/api/v1/users")

	// List all users (paginated)
	r.Get("/", c.findAll).
		Summary("List all users").
		Response(200, common.PaginatedResponse{})

	// Get own profile
	r.Get("/me", c.me).
		Summary("Get own profile").
		Response(200, UserPublic{})

	// Get user by ID
	r.Get("/:id", c.findOne).
		Summary("Get user by ID").
		Pipes(gonest.NewParseIntPipe("id")).
		Response(200, UserPublic{})

	// Update own profile
	r.Patch("/me", c.updateMe).
		Summary("Update own profile").
		Body(UpdateUserDto{}).
		Response(200, UserPublic{})

	// Create user (admin only)
	r.Post("/", c.create).
		Summary("Create a new user (admin)").
		Body(CreateUserDto{}).
		Response(201, UserPublic{}).
		SetMetadata("roles", []common.Role{common.RoleAdmin}).
		Guards(&common.RolesGuard{})

	// Update any user (admin only)
	r.Patch("/:id", c.updateAdmin).
		Summary("Update user by ID (admin)").
		Body(UpdateUserAdminDto{}).
		Pipes(gonest.NewParseIntPipe("id")).
		Response(200, UserPublic{}).
		SetMetadata("roles", []common.Role{common.RoleAdmin}).
		Guards(&common.RolesGuard{})

	// Delete user (admin only)
	r.Delete("/:id", c.remove).
		Summary("Delete user by ID (admin)").
		Pipes(gonest.NewParseIntPipe("id")).
		Response(204, nil).
		SetMetadata("roles", []common.Role{common.RoleAdmin}).
		Guards(&common.RolesGuard{})
}

func (c *UsersController) findAll(ctx gonest.Context) error {
	pq := common.NewPaginationQuery(ctx)
	users, total := c.service.FindAll(pq.Offset(), pq.Limit)

	publicUsers := make([]UserPublic, len(users))
	for i, u := range users {
		publicUsers[i] = u.ToPublic()
	}

	return ctx.JSON(http.StatusOK, common.NewPaginatedResponse(publicUsers, total, pq))
}

func (c *UsersController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	user := c.service.FindByID(id)
	if user == nil {
		return gonest.NewNotFoundException(fmt.Sprintf("user #%d not found", id))
	}
	return ctx.JSON(http.StatusOK, user.ToPublic())
}

func (c *UsersController) me(ctx gonest.Context) error {
	au, ok := ctx.Get("user")
	if !ok {
		return gonest.NewUnauthorizedException("not authenticated")
	}
	user := c.service.FindByID(au.(*common.AuthUser).ID)
	if user == nil {
		return gonest.NewNotFoundException("user not found")
	}
	return ctx.JSON(http.StatusOK, user.ToPublic())
}

func (c *UsersController) updateMe(ctx gonest.Context) error {
	au, ok := ctx.Get("user")
	if !ok {
		return gonest.NewUnauthorizedException("not authenticated")
	}
	var dto UpdateUserDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	user, err := c.service.Update(au.(*common.AuthUser).ID, dto)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, user.ToPublic())
}

func (c *UsersController) create(ctx gonest.Context) error {
	var dto CreateUserDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	user, err := c.service.Create(dto)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, user.ToPublic())
}

func (c *UsersController) updateAdmin(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	var dto UpdateUserAdminDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	user, err := c.service.UpdateAdmin(id, dto)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, user.ToPublic())
}

func (c *UsersController) remove(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	if err := c.service.Delete(id); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}
