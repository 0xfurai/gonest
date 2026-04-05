package articles

import "time"

// Article represents a blog article.
type Article struct {
	ID        int       `json:"id" swagger:"example=1"`
	Title     string    `json:"title" swagger:"example=Getting Started with GoNest"`
	Slug      string    `json:"slug" swagger:"example=getting-started-with-gonest"`
	Body      string    `json:"body" swagger:"example=GoNest brings NestJS architecture to Go..."`
	Summary   string    `json:"summary,omitempty" swagger:"example=An introduction to GoNest"`
	ImageURL  string    `json:"imageUrl,omitempty" swagger:"example=https://example.com/image.jpg"`
	AuthorID  int       `json:"authorId" swagger:"example=1"`
	Status    string    `json:"status" swagger:"example=published"`
	Tags      []string  `json:"tags"`
	ViewCount int       `json:"viewCount" swagger:"example=42"`
	CreatedAt time.Time `json:"createdAt" swagger:"format=date-time"`
	UpdatedAt time.Time `json:"updatedAt" swagger:"format=date-time"`
}

// ArticleWithAuthor includes author name for listing.
type ArticleWithAuthor struct {
	Article
	AuthorName string `json:"authorName"`
}
