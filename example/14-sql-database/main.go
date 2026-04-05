package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/0xfurai/gonest"
	gosql "github.com/0xfurai/gonest/database/sql"

	_ "modernc.org/sqlite"
)

// --- Entity ---

type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	IsActive  bool   `json:"isActive"`
}

// --- DTO ---

type CreateUserDto struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName"  validate:"required"`
}

// --- Service ---

type UsersService struct {
	db *sql.DB
}

func NewUsersService(db *sql.DB) *UsersService {
	// Run migrations
	err := gosql.Migrate(db, []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT 1
		)`,
	})
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	return &UsersService{db: db}
}

func (s *UsersService) Create(dto CreateUserDto) (*User, error) {
	result, err := s.db.Exec(
		"INSERT INTO users (first_name, last_name, is_active) VALUES (?, ?, ?)",
		dto.FirstName, dto.LastName, true,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &User{
		ID:        int(id),
		FirstName: dto.FirstName,
		LastName:  dto.LastName,
		IsActive:  true,
	}, nil
}

func (s *UsersService) FindAll() ([]User, error) {
	rows, err := s.db.Query("SELECT id, first_name, last_name, is_active FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.IsActive); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if users == nil {
		users = []User{}
	}
	return users, nil
}

func (s *UsersService) FindOne(id int) (*User, error) {
	var u User
	err := s.db.QueryRow(
		"SELECT id, first_name, last_name, is_active FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.FirstName, &u.LastName, &u.IsActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UsersService) Delete(id int) (bool, error) {
	result, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

// --- Controller ---

type UsersController struct {
	service *UsersService
}

func NewUsersController(service *UsersService) *UsersController {
	return &UsersController{service: service}
}

func (c *UsersController) Register(r gonest.Router) {
	r.Prefix("/users")

	r.Get("/", c.findAll)
	r.Get("/:id", c.findOne).Pipes(gonest.NewParseIntPipe("id"))
	r.Post("/", c.create).HttpCode(http.StatusCreated)
	r.Delete("/:id", c.remove).Pipes(gonest.NewParseIntPipe("id"))
}

func (c *UsersController) findAll(ctx gonest.Context) error {
	users, err := c.service.FindAll()
	if err != nil {
		return gonest.NewInternalServerError(err.Error())
	}
	return ctx.JSON(http.StatusOK, users)
}

func (c *UsersController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	user, err := c.service.FindOne(id)
	if err != nil {
		return gonest.NewInternalServerError(err.Error())
	}
	if user == nil {
		return gonest.NewNotFoundException(fmt.Sprintf("user #%d not found", id))
	}
	return ctx.JSON(http.StatusOK, user)
}

func (c *UsersController) create(ctx gonest.Context) error {
	var dto CreateUserDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	user, err := c.service.Create(dto)
	if err != nil {
		return gonest.NewInternalServerError(err.Error())
	}
	return ctx.JSON(http.StatusCreated, user)
}

func (c *UsersController) remove(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	deleted, err := c.service.Delete(id)
	if err != nil {
		return gonest.NewInternalServerError(err.Error())
	}
	if !deleted {
		return gonest.NewNotFoundException(fmt.Sprintf("user #%d not found", id))
	}
	return ctx.NoContent(http.StatusNoContent)
}

// --- Module ---

var DatabaseModule = gosql.NewModule(gosql.Options{
	Driver:   gosql.DriverSQLite,
	Database: "app.db",
})

var UsersModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewUsersController},
	Providers:   []any{NewUsersService},
	Exports:     []any{(*UsersService)(nil)},
})

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Imports: []*gonest.Module{DatabaseModule, UsersModule},
})

func main() {
	app := gonest.Create(AppModule)
	app.EnableCors()
	log.Fatal(app.Listen(":3000"))
}
