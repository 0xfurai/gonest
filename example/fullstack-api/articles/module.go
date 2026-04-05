package articles

import "github.com/0xfurai/gonest"

var Module = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewArticlesController},
	Providers:   []any{NewArticlesService}, // NewArticlesService(*sql.DB) — DB injected via global module
	Exports:     []any{(*ArticlesService)(nil)},
})
