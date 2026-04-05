// Package database provides shared database abstractions for GoNest.
//
// Sub-packages:
//   - database/sql  — Generic SQL database module (PostgreSQL, MySQL, SQLite, etc.)
//   - database/mongo — MongoDB module
package database

import "context"

// Repository is a generic interface for CRUD operations on any data store.
// Implement this with your ORM/driver of choice.
type Repository[T any] interface {
	FindAll(ctx context.Context) ([]T, error)
	FindByID(ctx context.Context, id any) (*T, error)
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id any) error
	Count(ctx context.Context) (int64, error)
}

// PaginatedResult wraps a query result with pagination metadata.
type PaginatedResult[T any] struct {
	Items      []T   `json:"items"`
	TotalCount int64 `json:"totalCount"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
}

// Paginate is a helper that applies offset/limit to a slice.
func Paginate[T any](items []T, page, limit int) PaginatedResult[T] {
	total := int64(len(items))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	if offset >= len(items) {
		return PaginatedResult[T]{Items: nil, TotalCount: total, Page: page, Limit: limit}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return PaginatedResult[T]{Items: items[offset:end], TotalCount: total, Page: page, Limit: limit}
}
