// Package main demonstrates a full-stack API built with GoNest.
//
// This example is the Go equivalent of https://github.com/brocoders/nestjs-boilerplate
// and showcases all major framework features:
//
//   - SQLite database with auto-migrations
//   - JWT Authentication (register, login, refresh tokens)
//   - Role-based authorization (admin/user roles)
//   - User management (CRUD with pagination)
//   - Article management (CRUD with filtering, slugs, view counts)
//   - File upload (with type/size validation, stored on disk + tracked in DB)
//   - API documentation (Swagger/OpenAPI)
//   - Health checks
//   - Rate limiting
//   - Request logging with request IDs
//   - CORS
//   - Mail service (stub)
//   - Response serialization (password exclusion)
//   - Global error handling
//   - API versioning (URI-based)
//   - Pagination
//   - Seed data (admin user + sample articles)
//
// Requirements:
//
//   go get modernc.org/sqlite   (pure Go SQLite driver, no CGO needed)
//
// Run:
//
//   go run .
//
// The database file is created at ./app.db.
// Delete it to reset all data.
//
// API:
//
//   # Register a user
//   curl -X POST http://localhost:3000/api/v1/auth/register \
//     -H 'Content-Type: application/json' \
//     -d '{"email":"user@test.com","password":"password123","firstName":"John","lastName":"Doe"}'
//
//   # Login as seeded admin
//   curl -X POST http://localhost:3000/api/v1/auth/login \
//     -H 'Content-Type: application/json' \
//     -d '{"email":"admin@example.com","password":"admin123"}'
//
//   # List articles (public, no auth needed)
//   curl http://localhost:3000/api/v1/articles
//
//   # Create article (needs auth)
//   curl -X POST http://localhost:3000/api/v1/articles \
//     -H 'Content-Type: application/json' \
//     -H 'Authorization: Bearer <token>' \
//     -d '{"title":"My Article","body":"Article content here...","tags":["go"]}'
//
//   # Upload a file (needs auth)
//   curl -X POST http://localhost:3000/api/v1/files/upload \
//     -H 'Authorization: Bearer <token>' \
//     -F 'file=@photo.jpg'
//
//   # Health check
//   curl http://localhost:3000/health
//
//   # Swagger docs
//   open http://localhost:3000/docs/
package main

import (
	"database/sql"
	"log"
	"time"

	// Pure-Go SQLite driver (no CGO). Replace with the import below for CGO version:
	// _ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/example/fullstack-api/articles"
	"github.com/0xfurai/gonest/example/fullstack-api/auth"
	"github.com/0xfurai/gonest/example/fullstack-api/common"
	"github.com/0xfurai/gonest/example/fullstack-api/files"
	"github.com/0xfurai/gonest/example/fullstack-api/mail"
	"github.com/0xfurai/gonest/example/fullstack-api/users"
	"github.com/0xfurai/gonest/health"
	"github.com/0xfurai/gonest/swagger"
)

func main() {
	logger := gonest.NewDefaultLogger()

	// --- Database ---

	db, err := InitDatabase("./app.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	dbModule := NewDatabaseModule(db)

	// Seed data
	seedDatabase(db, logger)

	// --- Health check with real DB ping ---

	healthModule := health.NewModule(health.Options{
		Indicators: []health.HealthIndicator{
			&health.PingIndicator{},
			&health.CustomIndicator{
				IndicatorName: "database",
				CheckFn: func() health.HealthResult {
					if err := db.Ping(); err != nil {
						return health.HealthResult{Status: health.StatusDown,
							Details: map[string]any{"error": err.Error()}}
					}
					stats := db.Stats()
					return health.HealthResult{Status: health.StatusUp, Details: map[string]any{
						"driver": "sqlite",
						"open":   stats.OpenConnections,
						"inUse":  stats.InUse,
					}}
				},
			},
		},
	})

	// --- Swagger ---

	swaggerModule := swagger.Module(swagger.Options{
		Title:       "GoNest Fullstack API",
		Description: "A full-featured REST API built with GoNest + SQLite",
		Version:     "1.0.0",
		Path:        "/docs",
		BearerAuth:  true,
	})

	// --- Root Module ---

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{
			dbModule,
			users.Module,
			auth.NewModule(),
			articles.Module,
			files.Module,
			healthModule,
			swaggerModule,
		},
		Providers: []any{
			gonest.ProvideValue[gonest.Logger](logger),
			func(logger gonest.Logger) *mail.MailService {
				return mail.NewMailService(logger)
			},
		},
	})

	// --- Application ---

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: logger})

	// Middleware
	app.UseGlobalMiddleware(
		common.NewRequestIDMiddleware(),
		common.NewRequestLoggerMiddleware(logger),
	)

	// CORS
	app.EnableCors(gonest.CorsOptions{
		Origin:      "*",
		Methods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		Headers:     "Content-Type, Authorization",
		Credentials: true,
	})

	// Global auth guard — constructed before Init() so it applies to all routes.
	// Uses the same *sql.DB as the DI-managed services.
	authUsersSvc := users.NewUsersService(db)
	authSvc := auth.NewAuthService(authUsersSvc)
	app.UseGlobalGuards(auth.NewJWTGuard(authSvc))

	// Rate limiting: 200 requests/min per IP
	app.UseGlobalGuards(gonest.NewThrottleGuard(200, time.Minute))

	// Validation
	app.UseGlobalPipes(gonest.NewValidationPipe())

	// Error handling
	app.UseGlobalFilters(&gonest.DefaultExceptionFilter{})

	// --- Start ---

	logger.Log("Database: ./app.db (SQLite)")
	log.Fatal(app.Listen(":3000"))
}

// seedDatabase populates the database with initial data.
func seedDatabase(db *sql.DB, logger gonest.Logger) {
	usersSvc := users.NewUsersService(db)
	if err := usersSvc.Seed(); err != nil {
		logger.Error("Failed to seed users: %v", err)
	}

	articlesSvc := articles.NewArticlesService(db)
	if err := articlesSvc.Seed(); err != nil {
		logger.Error("Failed to seed articles: %v", err)
	}

	logger.Log("Database seeded (admin@example.com / admin123)")
}
