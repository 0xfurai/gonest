package main

import (
	"log"
	"net/http"

	"github.com/0xfurai/gonest"
)

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password" serialize:"exclude"`
	SSN       string `json:"ssn" serialize:"group=admin"`
	Role      string `json:"role"`
}

var users = []User{
	{ID: 1, Name: "Alice", Email: "alice@example.com", Password: "hash1", SSN: "111-22-3333", Role: "admin"},
	{ID: 2, Name: "Bob", Email: "bob@example.com", Password: "hash2", SSN: "444-55-6666", Role: "user"},
}

type UsersController struct{}

func NewUsersController() *UsersController { return &UsersController{} }

func (c *UsersController) Register(r gonest.Router) {
	r.Prefix("/users")

	// List users (password excluded, SSN hidden)
	r.Get("/", c.findAll).
		Interceptors(gonest.NewSerializerInterceptor())

	// List users as admin (password excluded, SSN visible)
	r.Get("/admin", c.findAllAdmin).
		Interceptors(gonest.NewSerializerInterceptor()).
		SetMetadata("serialize_groups", []string{"admin"})
}

func (c *UsersController) findAll(ctx gonest.Context) error {
	// Store data for serializer interceptor to transform, then write
	ctx.Set("__serialize_data", users)
	ctx.Set("__serialize_status", http.StatusOK)
	return nil
}

func (c *UsersController) findAllAdmin(ctx gonest.Context) error {
	ctx.Set("__serialize_data", users)
	ctx.Set("__serialize_status", http.StatusOK)
	return nil
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewUsersController},
})

func main() {
	app := gonest.Create(AppModule)
	log.Fatal(app.Listen(":3000"))
}
