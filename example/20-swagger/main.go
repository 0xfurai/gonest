package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gonest"
	"github.com/gonest/swagger"
)

// --- DTOs ---

// CreateArticleDto represents the request body for creating an article.
type CreateArticleDto struct {
	Title   string `json:"title"   validate:"required" swagger:"example=My First Article"`
	Content string `json:"content" validate:"required" swagger:"example=Article body text..."`
	Author  string `json:"author"  validate:"required" swagger:"example=Jane Doe"`
}

// UpdateArticleDto represents the request body for updating an article.
type UpdateArticleDto struct {
	Title   string `json:"title,omitempty"   swagger:"example=Updated Title"`
	Content string `json:"content,omitempty" swagger:"example=Updated body text..."`
}

// Article represents a published article resource.
type Article struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`
}

// --- Service ---

// ArticlesService manages the in-memory article store.
type ArticlesService struct {
	mu       sync.RWMutex
	articles []Article
	nextID   int
}

// NewArticlesService creates a new ArticlesService.
func NewArticlesService() *ArticlesService {
	return &ArticlesService{
		nextID: 1,
		articles: []Article{
			{ID: 0, Title: "Welcome to GoNest", Content: "GoNest brings NestJS patterns to Go.", Author: "GoNest Team"},
		},
	}
}

// Create adds a new article and returns it.
func (s *ArticlesService) Create(dto CreateArticleDto) Article {
	s.mu.Lock()
	defer s.mu.Unlock()
	article := Article{
		ID:      s.nextID,
		Title:   dto.Title,
		Content: dto.Content,
		Author:  dto.Author,
	}
	s.nextID++
	s.articles = append(s.articles, article)
	return article
}

// FindAll returns all articles.
func (s *ArticlesService) FindAll() []Article {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Article, len(s.articles))
	copy(result, s.articles)
	return result
}

// FindOne returns an article by ID.
func (s *ArticlesService) FindOne(id int) *Article {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.articles {
		if a.ID == id {
			return &a
		}
	}
	return nil
}

// Update modifies an existing article.
func (s *ArticlesService) Update(id int, dto UpdateArticleDto) *Article {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.articles {
		if a.ID == id {
			if dto.Title != "" {
				s.articles[i].Title = dto.Title
			}
			if dto.Content != "" {
				s.articles[i].Content = dto.Content
			}
			updated := s.articles[i]
			return &updated
		}
	}
	return nil
}

// Delete removes an article by ID.
func (s *ArticlesService) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.articles {
		if a.ID == id {
			s.articles = append(s.articles[:i], s.articles[i+1:]...)
			return true
		}
	}
	return false
}

// --- Controller ---

// ArticlesController handles HTTP requests for article resources.
type ArticlesController struct {
	service *ArticlesService
}

// NewArticlesController creates a new ArticlesController.
func NewArticlesController(service *ArticlesService) *ArticlesController {
	return &ArticlesController{service: service}
}

// Register sets up routes with swagger annotations.
func (c *ArticlesController) Register(r gonest.Router) {
	r.Prefix("/articles")

	// GET /articles — List all articles (public)
	r.Get("/", c.findAll).
		Summary("List all articles").
		Tags("Articles").
		Response(http.StatusOK, []Article{}).
		SetMetadata("public", true)

	// GET /articles/:id — Get a single article by ID (public)
	r.Get("/:id", c.findOne).
		Summary("Get an article by ID").
		Tags("Articles").
		Response(http.StatusOK, Article{}).
		Pipes(gonest.NewParseIntPipe("id")).
		SetMetadata("public", true)

	// POST /articles — Create a new article (requires bearer auth)
	r.Post("/", c.create).
		Summary("Create a new article").
		Tags("Articles").
		Body(CreateArticleDto{}).
		Response(http.StatusCreated, Article{}).
		HttpCode(http.StatusCreated)

	// PUT /articles/:id — Update an article (requires bearer auth)
	r.Put("/:id", c.update).
		Summary("Update an existing article").
		Tags("Articles").
		Body(UpdateArticleDto{}).
		Response(http.StatusOK, Article{}).
		Pipes(gonest.NewParseIntPipe("id"))

	// DELETE /articles/:id — Delete an article (requires bearer auth)
	r.Delete("/:id", c.remove).
		Summary("Delete an article").
		Tags("Articles").
		Pipes(gonest.NewParseIntPipe("id"))
}

func (c *ArticlesController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.service.FindAll())
}

func (c *ArticlesController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	article := c.service.FindOne(id)
	if article == nil {
		return gonest.NewNotFoundException(fmt.Sprintf("article #%d not found", id))
	}
	return ctx.JSON(http.StatusOK, article)
}

func (c *ArticlesController) create(ctx gonest.Context) error {
	var dto CreateArticleDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	article := c.service.Create(dto)
	return ctx.JSON(http.StatusCreated, article)
}

func (c *ArticlesController) update(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	var dto UpdateArticleDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	article := c.service.Update(id, dto)
	if article == nil {
		return gonest.NewNotFoundException(fmt.Sprintf("article #%d not found", id))
	}
	return ctx.JSON(http.StatusOK, article)
}

func (c *ArticlesController) remove(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	if !c.service.Delete(id) {
		return gonest.NewNotFoundException(fmt.Sprintf("article #%d not found", id))
	}
	return ctx.NoContent(http.StatusNoContent)
}

// --- Module ---

// ArticlesModule bundles the articles controller and service.
var ArticlesModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewArticlesController},
	Providers:   []any{NewArticlesService},
	Exports:     []any{(*ArticlesService)(nil)},
})

// SwaggerModule configures OpenAPI documentation with bearer auth.
var SwaggerModule = swagger.Module(swagger.Options{
	Title:       "GoNest Articles API",
	Description: "A sample CRUD API for articles, showcasing Swagger/OpenAPI integration with GoNest.",
	Version:     "1.0.0",
	Path:        "/swagger",
	BearerAuth:  true,
})

// AppModule is the root application module.
var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Imports: []*gonest.Module{
		ArticlesModule,
		SwaggerModule,
	},
})

// --- Bootstrap ---

func main() {
	app := gonest.Create(AppModule)
	app.SetGlobalPrefix("/api")
	app.EnableCors()

	log.Println("Swagger UI available at http://localhost:3000/api/swagger")
	log.Fatal(app.Listen(":3000"))
}
