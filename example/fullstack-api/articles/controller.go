package articles

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gonest"
	"github.com/gonest/example/fullstack-api/common"
)

// ArticlesController handles /articles endpoints.
type ArticlesController struct {
	service *ArticlesService
}

func NewArticlesController(service *ArticlesService) *ArticlesController {
	return &ArticlesController{service: service}
}

func (c *ArticlesController) Register(r gonest.Router) {
	r.Prefix("/api/v1/articles")

	// List articles (public, filterable, paginated)
	r.Get("/", c.findAll).
		Summary("List articles").
		Response(200, common.PaginatedResponse{}).
		SetMetadata("public", true)

	// Get article by ID (public, increments view count)
	r.Get("/:id", c.findOne).
		Summary("Get article by ID").
		Pipes(gonest.NewParseIntPipe("id")).
		Response(200, Article{}).
		SetMetadata("public", true)

	// Create article (authenticated)
	r.Post("/", c.create).
		Summary("Create a new article").
		Body(CreateArticleDto{}).
		Response(201, Article{})

	// Update article (author only)
	r.Patch("/:id", c.update).
		Summary("Update an article").
		Body(UpdateArticleDto{}).
		Pipes(gonest.NewParseIntPipe("id")).
		Response(200, Article{})

	// Delete article (author or admin)
	r.Delete("/:id", c.remove).
		Summary("Delete an article").
		Pipes(gonest.NewParseIntPipe("id")).
		Response(204, nil)
}

func (c *ArticlesController) findAll(ctx gonest.Context) error {
	pq := common.NewPaginationQuery(ctx)
	filters := ArticleQueryDto{
		Status: ctx.Query("status"),
		Tag:    ctx.Query("tag"),
	}
	if aid := ctx.Query("authorId"); aid != "" {
		filters.AuthorID, _ = strconv.Atoi(aid)
	}

	articles, total := c.service.FindAll(pq.Offset(), pq.Limit, filters)
	return ctx.JSON(http.StatusOK, common.NewPaginatedResponse(articles, total, pq))
}

func (c *ArticlesController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	article := c.service.FindByID(id)
	if article == nil {
		return gonest.NewNotFoundException(fmt.Sprintf("article #%d not found", id))
	}
	c.service.IncrementViewCount(id)
	return ctx.JSON(http.StatusOK, article)
}

func (c *ArticlesController) create(ctx gonest.Context) error {
	user, ok := ctx.Get("user")
	if !ok {
		return gonest.NewUnauthorizedException("not authenticated")
	}
	var dto CreateArticleDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	article, err := c.service.Create(user.(*common.AuthUser).ID, dto)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, article)
}

func (c *ArticlesController) update(ctx gonest.Context) error {
	user, ok := ctx.Get("user")
	if !ok {
		return gonest.NewUnauthorizedException("not authenticated")
	}
	id := ctx.Param("id").(int)
	var dto UpdateArticleDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	article, err := c.service.Update(id, user.(*common.AuthUser).ID, dto)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, article)
}

func (c *ArticlesController) remove(ctx gonest.Context) error {
	user, ok := ctx.Get("user")
	if !ok {
		return gonest.NewUnauthorizedException("not authenticated")
	}
	id := ctx.Param("id").(int)
	au := user.(*common.AuthUser)
	isAdmin := au.Role == common.RoleAdmin
	if err := c.service.Delete(id, au.ID, isAdmin); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}
