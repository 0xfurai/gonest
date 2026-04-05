package gonest

// DiscoveryModule is a formal, importable module that provides DiscoveryService
// and GraphInspector via DI. Equivalent to NestJS DiscoveryModule.
//
// Usage:
//
//	appModule := gonest.NewModule(gonest.ModuleOptions{
//	    Imports: []*gonest.Module{
//	        gonest.NewDiscoveryModule(),
//	        // ...other modules
//	    },
//	    Controllers: []any{NewMyController},
//	})
//
// Then inject DiscoveryService or GraphInspector into any provider:
//
//	func NewMyService(ds *gonest.DiscoveryService) *MyService { ... }
func NewDiscoveryModule() *Module {
	return NewModule(ModuleOptions{
		Providers: []any{
			ProvideFactory[*DiscoveryService](func() *DiscoveryService {
				return NewDiscoveryService(NewReflector())
			}),
			ProvideFactory[*GraphInspector](func() *GraphInspector {
				return NewGraphInspector()
			}),
		},
		Exports: []any{
			(*DiscoveryService)(nil),
			(*GraphInspector)(nil),
		},
		Global: true,
	})
}
