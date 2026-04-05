package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gonest"
)

// --- DTOs ---

type CreateCatDto struct {
	Name  string `json:"name"  validate:"required"`
	Age   int    `json:"age"   validate:"required,gte=0"`
	Breed string `json:"breed" validate:"required"`
}

type Cat struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Breed string `json:"breed"`
}

// --- Service ---

type CatsService struct {
	mu    sync.RWMutex
	cats  []Cat
	nextID int
}

func NewCatsService() *CatsService {
	return &CatsService{nextID: 1}
}

func (s *CatsService) Create(dto CreateCatDto) Cat {
	s.mu.Lock()
	defer s.mu.Unlock()
	cat := Cat{
		ID:    s.nextID,
		Name:  dto.Name,
		Age:   dto.Age,
		Breed: dto.Breed,
	}
	s.nextID++
	s.cats = append(s.cats, cat)
	return cat
}

func (s *CatsService) FindAll() []Cat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Cat, len(s.cats))
	copy(result, s.cats)
	return result
}

func (s *CatsService) FindOne(id int) *Cat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.cats {
		if c.ID == id {
			return &c
		}
	}
	return nil
}

func (s *CatsService) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.cats {
		if c.ID == id {
			s.cats = append(s.cats[:i], s.cats[i+1:]...)
			return true
		}
	}
	return false
}

// --- Guard ---

type RolesGuard struct {
	reflector *gonest.Reflector
}

func NewRolesGuard(reflector *gonest.Reflector) *RolesGuard {
	return &RolesGuard{reflector: reflector}
}

func (g *RolesGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
	roles, ok := gonest.GetMetadata[[]string](ctx, "roles")
	if !ok {
		return true, nil
	}
	// In a real app, extract user from JWT/session
	userRole := ctx.Header("X-User-Role")
	if userRole == "" {
		return false, gonest.NewForbiddenException("no role provided")
	}
	for _, r := range roles {
		if r == userRole {
			return true, nil
		}
	}
	return false, gonest.NewForbiddenException(fmt.Sprintf("role %q not in %v", userRole, roles))
}

// --- Middleware ---

type LoggerMiddleware struct{}

func NewLoggerMiddleware() *LoggerMiddleware { return &LoggerMiddleware{} }

func (m *LoggerMiddleware) Use(ctx gonest.Context, next gonest.NextFunc) error {
	log.Printf("[%s] %s", ctx.Method(), ctx.Path())
	return next()
}

// --- Interceptor ---

type TimingInterceptor struct{}

func (i *TimingInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
	ctx.SetHeader("X-Powered-By", "GoNest")
	return next.Handle()
}

// --- Controller ---

type CatsController struct {
	service *CatsService
}

func NewCatsController(service *CatsService) *CatsController {
	return &CatsController{service: service}
}

func (c *CatsController) Register(r gonest.Router) {
	r.Prefix("/cats")
	r.UseInterceptors(&TimingInterceptor{})

	// List all cats
	r.Get("/", c.findAll)

	// Get cat by ID
	r.Get("/:id", c.findOne).
		Pipes(gonest.NewParseIntPipe("id"))

	// Create a new cat (admin only)
	r.Post("/", c.create).
		SetMetadata("roles", []string{"admin"}).
		Guards(NewRolesGuard(gonest.NewReflector())).
		HttpCode(http.StatusCreated)

	// Delete a cat (admin only)
	r.Delete("/:id", c.remove).
		Pipes(gonest.NewParseIntPipe("id")).
		SetMetadata("roles", []string{"admin"}).
		Guards(NewRolesGuard(gonest.NewReflector()))
}

func (c *CatsController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.service.FindAll())
}

func (c *CatsController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	cat := c.service.FindOne(id)
	if cat == nil {
		return gonest.NewNotFoundException(fmt.Sprintf("cat #%d not found", id))
	}
	return ctx.JSON(http.StatusOK, cat)
}

func (c *CatsController) create(ctx gonest.Context) error {
	var dto CreateCatDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	cat := c.service.Create(dto)
	return ctx.JSON(http.StatusCreated, cat)
}

func (c *CatsController) remove(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	if !c.service.Delete(id) {
		return gonest.NewNotFoundException(fmt.Sprintf("cat #%d not found", id))
	}
	return ctx.NoContent(http.StatusNoContent)
}

// --- Module ---

var CatsModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewCatsController},
	Providers:   []any{NewCatsService},
	Exports:     []any{(*CatsService)(nil)},
})

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Imports: []*gonest.Module{CatsModule},
})

// --- Bootstrap ---

func main() {
	app := gonest.Create(AppModule)
	app.UseGlobalMiddleware(NewLoggerMiddleware())
	app.EnableCors()

	log.Fatal(app.Listen(":3000"))
}
