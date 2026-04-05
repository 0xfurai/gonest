package articles

// CreateArticleDto is used when creating articles.
type CreateArticleDto struct {
	Title    string   `json:"title" validate:"required,min=3,max=200" swagger:"example=My First Article"`
	Body     string   `json:"body" validate:"required,min=10" swagger:"example=This is the article content..."`
	Summary  string   `json:"summary,omitempty" validate:"omitempty,max=500" swagger:"example=A brief summary"`
	ImageURL string   `json:"imageUrl,omitempty" validate:"omitempty,max=2000" swagger:"example=https://example.com/image.jpg"`
	Tags     []string `json:"tags,omitempty" swagger:"example=go"`
	Status   string   `json:"status,omitempty" validate:"omitempty,oneof=draft published" swagger:"example=draft"`
}

// UpdateArticleDto is used when updating articles.
type UpdateArticleDto struct {
	Title    string   `json:"title,omitempty" validate:"omitempty,min=3,max=200" swagger:"example=Updated Title"`
	Body     string   `json:"body,omitempty" validate:"omitempty,min=10" swagger:"example=Updated article content..."`
	Summary  string   `json:"summary,omitempty" validate:"omitempty,max=500" swagger:"example=Updated summary"`
	ImageURL string   `json:"imageUrl,omitempty" validate:"omitempty,max=2000" swagger:"example=https://example.com/new-image.jpg"`
	Tags     []string `json:"tags,omitempty"`
	Status   string   `json:"status,omitempty" validate:"omitempty,oneof=draft published" swagger:"example=published"`
}

// ArticleQueryDto filters article listings.
type ArticleQueryDto struct {
	Status   string `json:"status,omitempty"`
	AuthorID int    `json:"authorId,omitempty"`
	Tag      string `json:"tag,omitempty"`
}
