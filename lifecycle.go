package gonest

// OnModuleInit is called once the module's providers have been resolved.
type OnModuleInit interface {
	OnModuleInit() error
}

// OnModuleDestroy is called when the module is being torn down.
type OnModuleDestroy interface {
	OnModuleDestroy() error
}

// OnApplicationBootstrap is called once all modules have been initialized.
type OnApplicationBootstrap interface {
	OnApplicationBootstrap() error
}

// OnApplicationShutdown is called during graceful shutdown.
type OnApplicationShutdown interface {
	OnApplicationShutdown(signal string) error
}

// BeforeApplicationShutdown is called just before application shutdown begins.
type BeforeApplicationShutdown interface {
	BeforeApplicationShutdown(signal string) error
}
