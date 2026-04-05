package users

import "github.com/gonest"

var Module = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewUsersController},
	Providers:   []any{NewUsersService}, // NewUsersService(*sql.DB) — DB injected via global module
	Exports:     []any{(*UsersService)(nil)},
})
