package auth

import "github.com/gonest"

// NewModule creates the auth module. It requires UsersModule to be imported
// in the parent so that UsersService is available.
func NewModule() *gonest.Module {
	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{NewAuthController},
		Providers:   []any{NewAuthService},
		Exports:     []any{(*AuthService)(nil)},
	})
}
