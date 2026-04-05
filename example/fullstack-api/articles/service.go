package articles

import (
	"database/sql"
	"strings"
	"time"

	"github.com/gonest"
)

// ArticlesService handles article CRUD backed by a SQL database.
type ArticlesService struct {
	db *sql.DB
}

func NewArticlesService(db *sql.DB) *ArticlesService {
	return &ArticlesService{db: db}
}

// Seed inserts sample articles if the table is empty.
func (s *ArticlesService) Seed() error {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&count)
	if count > 0 {
		return nil
	}
	now := time.Now()
	seeds := []struct{ title, slug, body, summary, tags string }{
		{"Getting Started with GoNest", "getting-started-with-gonest",
			"GoNest brings NestJS architecture to Go. This guide walks you through building your first application.",
			"An introduction to the GoNest framework", "go,framework"},
		{"Building REST APIs in Go", "building-rest-apis-in-go",
			"Learn how to build production-ready REST APIs with controllers, services, guards, and pipes.",
			"REST API best practices", "go,rest,api"},
	}
	for _, a := range seeds {
		s.db.Exec(
			`INSERT INTO articles (title, slug, body, summary, author_id, status, tags, created_at, updated_at)
			 VALUES (?, ?, ?, ?, 1, 'published', ?, ?, ?)`,
			a.title, a.slug, a.body, a.summary, a.tags, now, now,
		)
	}
	return nil
}

func (s *ArticlesService) Create(authorID int, dto CreateArticleDto) (*Article, error) {
	status := dto.Status
	if status == "" {
		status = "draft"
	}
	now := time.Now()
	tags := strings.Join(dto.Tags, ",")

	result, err := s.db.Exec(
		`INSERT INTO articles (title, slug, body, summary, image_url, author_id, status, tags, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		dto.Title, slugify(dto.Title), dto.Body, dto.Summary, dto.ImageURL,
		authorID, status, tags, now, now,
	)
	if err != nil {
		return nil, gonest.NewInternalServerError("failed to create article: " + err.Error())
	}
	id, _ := result.LastInsertId()
	return s.FindByID(int(id)), nil
}

func (s *ArticlesService) FindAll(offset, limit int, filters ArticleQueryDto) ([]*Article, int) {
	where, args := buildWhere(filters)

	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM articles"+where, args...).Scan(&total)

	query := `SELECT id, title, slug, body, summary, image_url, author_id, status, tags, view_count, created_at, updated_at
	          FROM articles` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, total
	}
	defer rows.Close()

	var articles []*Article
	for rows.Next() {
		if a := scanArticleRows(rows); a != nil {
			articles = append(articles, a)
		}
	}
	return articles, total
}

func (s *ArticlesService) FindByID(id int) *Article {
	return scanArticleRow(s.db.QueryRow(
		`SELECT id, title, slug, body, summary, image_url, author_id, status, tags, view_count, created_at, updated_at
		 FROM articles WHERE id = ?`, id,
	))
}

func (s *ArticlesService) FindBySlug(slug string) *Article {
	return scanArticleRow(s.db.QueryRow(
		`SELECT id, title, slug, body, summary, image_url, author_id, status, tags, view_count, created_at, updated_at
		 FROM articles WHERE slug = ?`, slug,
	))
}

func (s *ArticlesService) Update(id, authorID int, dto UpdateArticleDto) (*Article, error) {
	existing := s.FindByID(id)
	if existing == nil {
		return nil, gonest.NewNotFoundException("article not found")
	}
	if existing.AuthorID != authorID {
		return nil, gonest.NewForbiddenException("not the author of this article")
	}

	var sets []string
	var args []any
	if dto.Title != "" {
		sets = append(sets, "title = ?", "slug = ?")
		args = append(args, dto.Title, slugify(dto.Title))
	}
	if dto.Body != "" {
		sets = append(sets, "body = ?")
		args = append(args, dto.Body)
	}
	if dto.Summary != "" {
		sets = append(sets, "summary = ?")
		args = append(args, dto.Summary)
	}
	if dto.ImageURL != "" {
		sets = append(sets, "image_url = ?")
		args = append(args, dto.ImageURL)
	}
	if dto.Tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, strings.Join(dto.Tags, ","))
	}
	if dto.Status != "" {
		sets = append(sets, "status = ?")
		args = append(args, dto.Status)
	}
	if len(sets) == 0 {
		return existing, nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now(), id)
	s.db.Exec("UPDATE articles SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	return s.FindByID(id), nil
}

func (s *ArticlesService) Delete(id, authorID int, isAdmin bool) error {
	if !isAdmin {
		existing := s.FindByID(id)
		if existing == nil {
			return gonest.NewNotFoundException("article not found")
		}
		if existing.AuthorID != authorID {
			return gonest.NewForbiddenException("not the author of this article")
		}
	}
	result, err := s.db.Exec("DELETE FROM articles WHERE id = ?", id)
	if err != nil {
		return gonest.NewInternalServerError("delete failed: " + err.Error())
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return gonest.NewNotFoundException("article not found")
	}
	return nil
}

func (s *ArticlesService) IncrementViewCount(id int) {
	s.db.Exec("UPDATE articles SET view_count = view_count + 1 WHERE id = ?", id)
}

// --- helpers ---

func buildWhere(f ArticleQueryDto) (string, []any) {
	var conds []string
	var args []any
	if f.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, f.Status)
	}
	if f.AuthorID != 0 {
		conds = append(conds, "author_id = ?")
		args = append(args, f.AuthorID)
	}
	if f.Tag != "" {
		conds = append(conds, "(tags LIKE ? OR tags LIKE ? OR tags LIKE ? OR tags = ?)")
		args = append(args, f.Tag+",%", "%,"+f.Tag+",%", "%,"+f.Tag, f.Tag)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func slugify(title string) string {
	slug := strings.ToLower(title)
	var buf []byte
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			buf = append(buf, byte(c))
		} else if c == ' ' || c == '-' {
			if len(buf) > 0 && buf[len(buf)-1] != '-' {
				buf = append(buf, '-')
			}
		}
	}
	return strings.TrimRight(string(buf), "-")
}

func scanArticleRows(rows *sql.Rows) *Article {
	a := &Article{}
	var tags string
	if err := rows.Scan(&a.ID, &a.Title, &a.Slug, &a.Body, &a.Summary, &a.ImageURL,
		&a.AuthorID, &a.Status, &tags, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil
	}
	if tags != "" {
		a.Tags = strings.Split(tags, ",")
	}
	return a
}

func scanArticleRow(row *sql.Row) *Article {
	a := &Article{}
	var tags string
	if err := row.Scan(&a.ID, &a.Title, &a.Slug, &a.Body, &a.Summary, &a.ImageURL,
		&a.AuthorID, &a.Status, &tags, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil
	}
	if tags != "" {
		a.Tags = strings.Split(tags, ",")
	}
	return a
}
