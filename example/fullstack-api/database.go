package main

import (
	"database/sql"

	"github.com/0xfurai/gonest"
)

// Migrations defines the database schema.
var Migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		first_name TEXT NOT NULL DEFAULT '',
		last_name TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT 'user',
		status TEXT NOT NULL DEFAULT 'active',
		avatar_url TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		slug TEXT UNIQUE NOT NULL,
		body TEXT NOT NULL,
		summary TEXT NOT NULL DEFAULT '',
		image_url TEXT NOT NULL DEFAULT '',
		author_id INTEGER NOT NULL REFERENCES users(id),
		status TEXT NOT NULL DEFAULT 'draft',
		tags TEXT NOT NULL DEFAULT '',
		view_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL,
		original_name TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		size INTEGER NOT NULL,
		url TEXT NOT NULL,
		uploader_id INTEGER NOT NULL REFERENCES users(id),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
	`CREATE INDEX IF NOT EXISTS idx_articles_slug ON articles(slug)`,
	`CREATE INDEX IF NOT EXISTS idx_articles_author ON articles(author_id)`,
	`CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status)`,
}

// InitDatabase opens a SQLite database and runs migrations.
func InitDatabase(path string) (*sql.DB, error) {
	// "sqlite" is the driver name for modernc.org/sqlite (pure Go, no CGO).
	// Use "sqlite3" if using github.com/mattn/go-sqlite3 (CGO).
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Enable WAL mode for better concurrency
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")

	for _, stmt := range Migrations {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

// NewDatabaseModule creates a gonest module that provides *sql.DB.
func NewDatabaseModule(db *sql.DB) *gonest.Module {
	return gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*sql.DB](db)},
		Exports:   []any{(*sql.DB)(nil)},
		Global:    true,
	})
}
