package gonest

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/gonest/microservice"
)

// ApplicationContext is a standalone application context without HTTP.
// Useful for CLI tools, workers, cron jobs, or microservice-only apps.
// Equivalent to NestJS INestApplicationContext.
type ApplicationContext struct {
	module    *Module
	logger    Logger
	reflector *Reflector

	discovery      *DiscoveryService
	graphInspector *GraphInspector
}

// CreateApplicationContext bootstraps the DI container and module tree
// without starting an HTTP server. Useful for CLI tools, workers, or tests.
// Equivalent to NestJS NestFactory.createApplicationContext().
func CreateApplicationContext(rootModule *Module, opts ...ApplicationOptions) (*ApplicationContext, error) {
	var opt ApplicationOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	logger := opt.Logger
	if logger == nil {
		logger = NewDefaultLogger()
	}

	reflector := NewReflector()
	ctx := &ApplicationContext{
		module:         rootModule,
		logger:         logger,
		reflector:      reflector,
		discovery:      NewDiscoveryService(reflector),
		graphInspector: NewGraphInspector(),
	}

	if err := ctx.init(); err != nil {
		return nil, err
	}

	return ctx, nil
}

func (ctx *ApplicationContext) init() error {
	if err := ctx.module.compile(nil, ctx.logger, ctx.reflector); err != nil {
		return fmt.Errorf("gonest: module compilation failed: %w", err)
	}

	allMods := ctx.module.allModules()
	ctx.discovery.SetModules(allMods)
	ctx.graphInspector.SetModules(allMods)

	// Run OnApplicationBootstrap hooks
	for _, mod := range allMods {
		if mod.container == nil {
			continue
		}
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			if hook, ok := inst.(OnApplicationBootstrap); ok {
				if err := hook.OnApplicationBootstrap(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GetContainer returns the root module's DI container.
func (ctx *ApplicationContext) GetContainer() *Container {
	return ctx.module.container
}

// Resolve resolves a type from the root container.
func (ctx *ApplicationContext) Resolve(t reflect.Type) (any, error) {
	return ctx.module.container.Resolve(t)
}

// Close gracefully shuts down the application context.
func (ctx *ApplicationContext) Close() error {
	for _, mod := range ctx.module.allModules() {
		if mod.container == nil {
			continue
		}
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			if hook, ok := inst.(BeforeApplicationShutdown); ok {
				_ = hook.BeforeApplicationShutdown("")
			}
		}
	}

	for _, mod := range ctx.module.allModules() {
		if mod.container == nil {
			continue
		}
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			if hook, ok := inst.(OnApplicationShutdown); ok {
				_ = hook.OnApplicationShutdown("")
			}
		}
	}

	return ctx.module.destroy()
}

// GetDiscoveryService returns the discovery service.
func (ctx *ApplicationContext) GetDiscoveryService() *DiscoveryService {
	return ctx.discovery
}

// GetGraphInspector returns the graph inspector.
func (ctx *ApplicationContext) GetGraphInspector() *GraphInspector {
	return ctx.graphInspector
}

// MicroserviceApp wraps a microservice server with the DI module system.
// Equivalent to NestJS INestMicroservice.
type MicroserviceApp struct {
	ApplicationContext
	server microservice.Server
}

// MicroserviceOptions configures a microservice application.
type MicroserviceOptions struct {
	Logger Logger
	Server microservice.Server
}

// CreateMicroservice bootstraps a microservice application.
// Equivalent to NestJS NestFactory.createMicroservice().
func CreateMicroservice(rootModule *Module, opts MicroserviceOptions) (*MicroserviceApp, error) {
	logger := opts.Logger
	if logger == nil {
		logger = NewDefaultLogger()
	}

	reflector := NewReflector()
	app := &MicroserviceApp{
		ApplicationContext: ApplicationContext{
			module:         rootModule,
			logger:         logger,
			reflector:      reflector,
			discovery:      NewDiscoveryService(reflector),
			graphInspector: NewGraphInspector(),
		},
		server: opts.Server,
	}

	if err := app.init(); err != nil {
		return nil, err
	}

	return app, nil
}

// Listen starts the microservice server.
func (app *MicroserviceApp) Listen() error {
	app.logger.Log("Microservice is listening")
	return app.server.Listen()
}

// ListenWithGracefulShutdown starts the microservice and handles OS signals.
func (app *MicroserviceApp) ListenWithGracefulShutdown() error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.server.Listen()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		app.logger.Log("Received signal %v, shutting down microservice...", sig)
		return app.Close()
	}
}

// GetServer returns the underlying microservice server.
func (app *MicroserviceApp) GetServer() microservice.Server {
	return app.server
}

// Close shuts down the microservice.
func (app *MicroserviceApp) Close() error {
	if err := app.server.Close(); err != nil {
		return err
	}
	return app.ApplicationContext.Close()
}
