package common

import (
	"strconv"

	"github.com/gonest"
)

// PaginationQuery extracts pagination from query params.
type PaginationQuery struct {
	Page    int `json:"page"`
	Limit   int `json:"limit"`
	offset  int
}

func NewPaginationQuery(ctx gonest.Context) PaginationQuery {
	page, _ := strconv.Atoi(ctx.Query("page"))
	limit, _ := strconv.Atoi(ctx.Query("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return PaginationQuery{
		Page:   page,
		Limit:  limit,
		offset: (page - 1) * limit,
	}
}

func (p PaginationQuery) Offset() int { return p.offset }

// PaginatedResponse wraps a list with pagination metadata.
type PaginatedResponse struct {
	Data       any  `json:"data"`
	Page       int  `json:"page" swagger:"example=1"`
	Limit      int  `json:"limit" swagger:"example=10"`
	TotalCount int  `json:"totalCount" swagger:"example=42"`
	TotalPages int  `json:"totalPages" swagger:"example=5"`
	HasMore    bool `json:"hasMore" swagger:"example=true"`
}

func NewPaginatedResponse(data any, totalCount int, query PaginationQuery) PaginatedResponse {
	totalPages := totalCount / query.Limit
	if totalCount%query.Limit > 0 {
		totalPages++
	}
	return PaginatedResponse{
		Data:       data,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasMore:    query.Page < totalPages,
	}
}
