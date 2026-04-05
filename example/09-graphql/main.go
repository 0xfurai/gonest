package main

import (
	"log"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/graphql"
)

type Recipe struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Ingredients []string `json:"ingredients"`
}

var recipes = []Recipe{
	{ID: 1, Title: "Pizza", Description: "Classic Margherita", Ingredients: []string{"dough", "tomato", "mozzarella"}},
	{ID: 2, Title: "Pasta", Description: "Carbonara", Ingredients: []string{"spaghetti", "eggs", "guanciale", "pecorino"}},
}

func main() {
	engine := graphql.NewEngine()

	engine.Query("recipes", func(ctx *graphql.ResolverContext) (any, error) {
		return recipes, nil
	})

	engine.Query("recipe", func(ctx *graphql.ResolverContext) (any, error) {
		// In a real app, parse args from the query
		return recipes[0], nil
	})

	engine.Mutation("addRecipe", func(ctx *graphql.ResolverContext) (any, error) {
		recipe := Recipe{
			ID:    len(recipes) + 1,
			Title: "New Recipe",
		}
		recipes = append(recipes, recipe)
		return recipe, nil
	})

	gqlModule := graphql.NewModule(graphql.Options{
		Path:       "/graphql",
		Playground: true,
	}, engine)

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{gqlModule},
	})

	app := gonest.Create(appModule)
	log.Fatal(app.Listen(":3000"))
}
