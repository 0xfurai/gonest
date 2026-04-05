package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/database/mongo"
)

// --- DTOs ---

// CreateUserDto represents the request body for creating a user.
type CreateUserDto struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required"`
	Age   int    `json:"age"   validate:"required,gte=0"`
}

// UpdateUserDto represents the request body for updating a user.
type UpdateUserDto struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Age   int    `json:"age,omitempty"`
}

// User represents a user document stored in MongoDB.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// --- Schema ---

// UserSchema defines the MongoDB collection and indexes for users.
var UserSchema = mongo.Schema{
	Collection: "users",
	Indexes: []mongo.Index{
		{Fields: []string{"email"}, Unique: true},
	},
}

// --- Service ---

// UsersService provides CRUD operations for users.
// In a real application, this would use the mongo.Connection to interact
// with MongoDB. Here we simulate it with an in-memory store to demonstrate
// the wiring pattern.
type UsersService struct {
	conn   *mongo.Connection
	mu     sync.RWMutex
	users  []User
	nextID int
}

// NewUsersService creates a new UsersService with the MongoDB connection injected.
func NewUsersService(conn *mongo.Connection) *UsersService {
	log.Printf("UsersService initialized with MongoDB at %s (database: %s)",
		conn.URI, conn.Database)
	return &UsersService{
		conn:   conn,
		nextID: 1,
	}
}

// Create adds a new user document.
func (s *UsersService) Create(dto CreateUserDto) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := User{
		ID:    fmt.Sprintf("%06x", s.nextID),
		Name:  dto.Name,
		Email: dto.Email,
		Age:   dto.Age,
	}
	s.nextID++
	s.users = append(s.users, user)
	return user
}

// FindAll returns all user documents.
func (s *UsersService) FindAll() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]User, len(s.users))
	copy(result, s.users)
	return result
}

// FindOne returns a user by ID.
func (s *UsersService) FindOne(id string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.ID == id {
			return &u
		}
	}
	return nil
}

// Update modifies an existing user document.
func (s *UsersService) Update(id string, dto UpdateUserDto) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, u := range s.users {
		if u.ID == id {
			if dto.Name != "" {
				s.users[i].Name = dto.Name
			}
			if dto.Email != "" {
				s.users[i].Email = dto.Email
			}
			if dto.Age > 0 {
				s.users[i].Age = dto.Age
			}
			updated := s.users[i]
			return &updated
		}
	}
	return nil
}

// Delete removes a user document by ID.
func (s *UsersService) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, u := range s.users {
		if u.ID == id {
			s.users = append(s.users[:i], s.users[i+1:]...)
			return true
		}
	}
	return false
}

// --- Controller ---

// UsersController handles HTTP requests for the users resource.
type UsersController struct {
	service *UsersService
}

// NewUsersController creates a new UsersController.
func NewUsersController(service *UsersService) *UsersController {
	return &UsersController{service: service}
}

// Register defines routes for the users controller.
func (c *UsersController) Register(r gonest.Router) {
	r.Prefix("/users")

	r.Get("/", c.findAll)
	r.Get("/:id", c.findOne)
	r.Post("/", c.create).HttpCode(http.StatusCreated)
	r.Put("/:id", c.update)
	r.Delete("/:id", c.remove)
}

func (c *UsersController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.service.FindAll())
}

func (c *UsersController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id").(string)
	user := c.service.FindOne(id)
	if user == nil {
		return gonest.NewNotFoundException(fmt.Sprintf("user %q not found", id))
	}
	return ctx.JSON(http.StatusOK, user)
}

func (c *UsersController) create(ctx gonest.Context) error {
	var dto CreateUserDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	user := c.service.Create(dto)
	return ctx.JSON(http.StatusCreated, user)
}

func (c *UsersController) update(ctx gonest.Context) error {
	id := ctx.Param("id").(string)
	var dto UpdateUserDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	user := c.service.Update(id, dto)
	if user == nil {
		return gonest.NewNotFoundException(fmt.Sprintf("user %q not found", id))
	}
	return ctx.JSON(http.StatusOK, user)
}

func (c *UsersController) remove(ctx gonest.Context) error {
	id := ctx.Param("id").(string)
	if !c.service.Delete(id) {
		return gonest.NewNotFoundException(fmt.Sprintf("user %q not found", id))
	}
	return ctx.NoContent(http.StatusNoContent)
}

// --- Module ---

// MongoModule configures the MongoDB connection.
// In production, pass the real URI via environment variables.
var MongoModule = mongo.NewModule(mongo.Options{
	URI:      "mongodb://localhost:27017",
	Database: "gonest_example",
})

// UsersModule bundles the users controller and service.
// The UsersService depends on *mongo.Connection, which is exported
// globally by the MongoModule.
var UsersModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewUsersController},
	Providers:   []any{NewUsersService},
	Exports:     []any{(*UsersService)(nil)},
})

// AppModule is the root application module.
var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Imports: []*gonest.Module{
		MongoModule,
		UsersModule,
	},
})

// --- Bootstrap ---

func main() {
	app := gonest.Create(AppModule)
	app.EnableCors()

	log.Println("MongoDB example running at http://localhost:3000")
	log.Println("Endpoints: GET/POST /users, GET/PUT/DELETE /users/:id")
	log.Fatal(app.Listen(":3000"))
}
