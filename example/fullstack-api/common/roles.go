package common

import "github.com/gonest"

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// RolesGuard checks that the authenticated user has the required role.
type RolesGuard struct{}

func (g *RolesGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
	requiredRoles, ok := gonest.GetMetadata[[]Role](ctx, "roles")
	if !ok {
		return true, nil
	}

	userVal, ok := ctx.Get("user")
	if !ok {
		return false, gonest.NewForbiddenException("no user in context")
	}
	user := userVal.(*AuthUser)

	for _, required := range requiredRoles {
		if user.Role == required {
			return true, nil
		}
	}
	return false, gonest.NewForbiddenException("insufficient permissions")
}

// AuthUser is the user data attached to the request after JWT verification.
type AuthUser struct {
	ID    int    `json:"id" swagger:"example=1"`
	Email string `json:"email" swagger:"example=admin@example.com"`
	Role  Role   `json:"role" swagger:"example=admin"`
}
