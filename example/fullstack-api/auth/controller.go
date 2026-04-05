package auth

import (
	"net/http"

	"github.com/gonest"
	"github.com/gonest/example/fullstack-api/common"
)

// AuthController handles /auth endpoints.
type AuthController struct {
	service *AuthService
}

func NewAuthController(service *AuthService) *AuthController {
	return &AuthController{service: service}
}

func (c *AuthController) Register(r gonest.Router) {
	r.Prefix("/api/v1/auth")

	// Register a new user (public)
	r.Post("/register", c.register).
		Summary("Register a new user account").
		Body(RegisterDto{}).
		Response(201, TokenResponse{}).
		SetMetadata("public", true)

	// Login with email + password (public)
	r.Post("/login", c.login).
		Summary("Login with email and password").
		Body(LoginDto{}).
		Response(200, TokenResponse{}).
		SetMetadata("public", true)

	// Refresh access token (public)
	r.Post("/refresh", c.refresh).
		Summary("Refresh access token").
		Body(RefreshDto{}).
		Response(200, TokenResponse{}).
		SetMetadata("public", true)

	// Get current authenticated user
	r.Get("/me", c.me).
		Summary("Get current user info").
		Response(200, common.AuthUser{})
}

func (c *AuthController) register(ctx gonest.Context) error {
	var dto RegisterDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	tokens, err := c.service.Register(dto)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, tokens)
}

func (c *AuthController) login(ctx gonest.Context) error {
	var dto LoginDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	tokens, err := c.service.Login(dto)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, tokens)
}

func (c *AuthController) refresh(ctx gonest.Context) error {
	var dto RefreshDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	tokens, err := c.service.RefreshTokens(dto.RefreshToken)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, tokens)
}

func (c *AuthController) me(ctx gonest.Context) error {
	user, ok := ctx.Get("user")
	if !ok {
		return gonest.NewUnauthorizedException("not authenticated")
	}
	return ctx.JSON(http.StatusOK, user.(*common.AuthUser))
}
