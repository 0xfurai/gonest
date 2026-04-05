package config

import "github.com/0xfurai/gonest"

// ModuleOptions configures the config module.
type ModuleOptions struct {
	// EnvFilePath is the path to a .env file. If empty, only OS env vars are used.
	EnvFilePath string
	// IsGlobal makes the config module globally available.
	IsGlobal bool
}

// NewModule creates a config module that provides ConfigService.
func NewModule(opts ...ModuleOptions) *gonest.Module {
	var opt ModuleOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	svc := NewConfigService(opt.EnvFilePath)

	return gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*ConfigService](svc)},
		Exports:   []any{(*ConfigService)(nil)},
		Global:    opt.IsGlobal,
	})
}
