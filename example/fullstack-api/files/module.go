package files

import "github.com/0xfurai/gonest"

var Module = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewFilesController},
	Providers:   []any{NewFilesService},
	Exports:     []any{(*FilesService)(nil)},
})
