package articles

import "github.com/gonest"

var Module = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewArticlesController},
	Providers:   []any{NewArticlesService}, // NewArticlesService(*sql.DB) — DB injected via global module
	Exports:     []any{(*ArticlesService)(nil)},
})
