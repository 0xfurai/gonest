package auth

import (
	"strings"

	"github.com/gonest"
)

// JWTGuard is the global authentication guard.
// Routes marked with metadata "public"=true skip authentication.
type JWTGuard struct {
	authService *AuthService
}

func NewJWTGuard(authService *AuthService) *JWTGuard {
	return &JWTGuard{authService: authService}
}

func (g *JWTGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
	// Check if route is public
	if isPublic, ok := gonest.GetMetadata[bool](ctx, "public"); ok && isPublic {
		return true, nil
	}

	// Extract Bearer token
	auth := ctx.Header("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false, gonest.NewUnauthorizedException("missing or invalid Authorization header")
	}
	token := strings.TrimPrefix(auth, "Bearer ")

	// Validate token
	user, err := g.authService.ValidateAccessToken(token)
	if err != nil {
		return false, err
	}

	// Attach user to context
	ctx.Set("user", user)
	return true, nil
}
