// Package sql provides a generic SQL database module for GoNest.
//
// It supports any database/sql compatible driver — PostgreSQL, MySQL, SQLite,
// SQL Server, CockroachDB, etc. Just import the driver in your application:
//
//	import _ "github.com/lib/pq"           // PostgreSQL
//	import _ "github.com/go-sql-driver/mysql" // MySQL
//	import _ "modernc.org/sqlite"          // SQLite (pure Go)
//	import _ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL (pgx)
package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gonest"
)

// Driver identifies the SQL database driver.
type Driver string

const (
	DriverPostgres  Driver = "postgres"
	DriverMySQL     Driver = "mysql"
	DriverSQLite    Driver = "sqlite"
	DriverSQLServer Driver = "sqlserver"
)

// Options configures the SQL database connection.
type Options struct {
	// Driver is the database/sql driver name (e.g., "postgres", "mysql", "sqlite").
	Driver Driver

	// DSN is the full data source name / connection string.
	// If set, Host/Port/User/Password/Database are ignored.
	DSN string

	// Individual connection parameters (used when DSN is empty).
	Host     string
	Port     int
	User     string
	Password string
	Database string

	// Connection pool settings.
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// Driver-specific options.
	SSLMode  string            // PostgreSQL: "disable", "require", etc.
	Params   map[string]string // Additional connection parameters.
}

// BuildDSN generates a connection string from the individual parameters.
func (o Options) BuildDSN() string {
	if o.DSN != "" {
		return o.DSN
	}

	switch o.Driver {
	case DriverPostgres:
		return o.buildPostgresDSN()
	case DriverMySQL:
		return o.buildMySQLDSN()
	case DriverSQLite:
		return o.buildSQLiteDSN()
	case DriverSQLServer:
		return o.buildSQLServerDSN()
	default:
		return o.buildPostgresDSN()
	}
}

func (o Options) buildPostgresDSN() string {
	port := o.Port
	if port == 0 {
		port = 5432
	}
	sslMode := o.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		o.Host, port, o.User, o.Password, o.Database, sslMode)
	for k, v := range o.Params {
		dsn += fmt.Sprintf(" %s=%s", k, v)
	}
	return dsn
}

func (o Options) buildMySQLDSN() string {
	port := o.Port
	if port == 0 {
		port = 3306
	}
	// user:password@tcp(host:port)/database?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		o.User, o.Password, o.Host, port, o.Database)
	var params []string
	params = append(params, "parseTime=true")
	for k, v := range o.Params {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}
	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}
	return dsn
}

func (o Options) buildSQLiteDSN() string {
	if o.Database == "" {
		return ":memory:"
	}
	dsn := o.Database
	if len(o.Params) > 0 {
		var params []string
		for k, v := range o.Params {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		dsn += "?" + strings.Join(params, "&")
	}
	return dsn
}

func (o Options) buildSQLServerDSN() string {
	port := o.Port
	if port == 0 {
		port = 1433
	}
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
		o.User, o.Password, o.Host, port, o.Database)
}

// NewModule creates a database module that provides *sql.DB via DI.
// The module registers the DB globally so all modules can inject it.
//
// Usage:
//
//	dbModule := sql.NewModule(sql.Options{
//	    Driver:   sql.DriverPostgres,
//	    Host:     "localhost",
//	    Port:     5432,
//	    User:     "myuser",
//	    Password: "mypass",
//	    Database: "mydb",
//	})
func NewModule(opts Options) *gonest.Module {
	return gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideFactory[*sql.DB](func() (*sql.DB, error) {
				db, err := sql.Open(string(opts.Driver), opts.BuildDSN())
				if err != nil {
					return nil, fmt.Errorf("gonest/database/sql: open failed: %w", err)
				}
				if opts.MaxOpenConns > 0 {
					db.SetMaxOpenConns(opts.MaxOpenConns)
				}
				if opts.MaxIdleConns > 0 {
					db.SetMaxIdleConns(opts.MaxIdleConns)
				}
				if opts.ConnMaxLifetime > 0 {
					db.SetConnMaxLifetime(opts.ConnMaxLifetime)
				}
				if opts.ConnMaxIdleTime > 0 {
					db.SetConnMaxIdleTime(opts.ConnMaxIdleTime)
				}
				return db, nil
			}),
		},
		Exports: []any{(*sql.DB)(nil)},
		Global:  true,
	})
}

// NewModuleFromDSN creates a database module from a raw DSN string.
//
//	dbModule := sql.NewModuleFromDSN("postgres", "postgres://user:pass@host/db?sslmode=disable")
func NewModuleFromDSN(driver Driver, dsn string) *gonest.Module {
	return NewModule(Options{Driver: driver, DSN: dsn})
}

// HealthChecker verifies database connectivity. Use with the health module.
type HealthChecker struct {
	db   *sql.DB
	name string
}

// NewHealthChecker creates a health indicator for this database.
func NewHealthChecker(db *sql.DB, name string) *HealthChecker {
	if name == "" {
		name = "database"
	}
	return &HealthChecker{db: db, name: name}
}

func (h *HealthChecker) Name() string { return h.name }

// Check pings the database and reports status.
func (h *HealthChecker) Check() map[string]any {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		return map[string]any{"status": "down", "error": err.Error()}
	}

	stats := h.db.Stats()
	return map[string]any{
		"status":      "up",
		"openConns":   stats.OpenConnections,
		"inUse":       stats.InUse,
		"idle":        stats.Idle,
		"maxOpen":     stats.MaxOpenConnections,
	}
}

// Migrate runs a list of SQL migration statements in order.
// For production, use a dedicated migration tool (goose, migrate, atlas).
func Migrate(db *sql.DB, statements []string) error {
	for i, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}
	return nil
}

// Transaction runs fn inside a database transaction.
// If fn returns an error, the transaction is rolled back.
func Transaction(db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
