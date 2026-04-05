package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/gonest"
	"github.com/gonest/graphql"
)

// --- Users Service Data ---

// User represents a user entity in the users service.
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// UsersStore is the in-memory store for the users service.
type UsersStore struct {
	mu     sync.RWMutex
	users  []User
	nextID int
}

// NewUsersStore creates a UsersStore with sample data.
func NewUsersStore() *UsersStore {
	return &UsersStore{
		nextID: 4,
		users: []User{
			{ID: 1, Name: "Alice Johnson", Email: "alice@example.com", Username: "alice"},
			{ID: 2, Name: "Bob Smith", Email: "bob@example.com", Username: "bob"},
			{ID: 3, Name: "Charlie Brown", Email: "charlie@example.com", Username: "charlie"},
		},
	}
}

// FindAll returns all users.
func (s *UsersStore) FindAll() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]User, len(s.users))
	copy(result, s.users)
	return result
}

// FindByID returns a user by ID.
func (s *UsersStore) FindByID(id int) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.ID == id {
			return &u
		}
	}
	return nil
}

// Create adds a new user.
func (s *UsersStore) Create(name, email, username string) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := User{ID: s.nextID, Name: name, Email: email, Username: username}
	s.nextID++
	s.users = append(s.users, u)
	return u
}

// --- Reviews Service Data ---

// Review represents a review entity in the reviews service.
type Review struct {
	ID       int    `json:"id"`
	Body     string `json:"body"`
	Rating   int    `json:"rating"`
	AuthorID int    `json:"authorId"`
	Product  string `json:"product"`
}

// ReviewsStore is the in-memory store for the reviews service.
type ReviewsStore struct {
	mu      sync.RWMutex
	reviews []Review
	nextID  int
}

// NewReviewsStore creates a ReviewsStore with sample data.
func NewReviewsStore() *ReviewsStore {
	return &ReviewsStore{
		nextID: 5,
		reviews: []Review{
			{ID: 1, Body: "Excellent product, highly recommend!", Rating: 5, AuthorID: 1, Product: "GoNest Framework"},
			{ID: 2, Body: "Good but could use more documentation.", Rating: 4, AuthorID: 2, Product: "GoNest Framework"},
			{ID: 3, Body: "Fast and reliable, great for production.", Rating: 5, AuthorID: 1, Product: "GoNest CLI"},
			{ID: 4, Body: "Decent experience overall.", Rating: 3, AuthorID: 3, Product: "GoNest CLI"},
		},
	}
}

// FindAll returns all reviews.
func (s *ReviewsStore) FindAll() []Review {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Review, len(s.reviews))
	copy(result, s.reviews)
	return result
}

// FindByAuthor returns all reviews by a specific author.
func (s *ReviewsStore) FindByAuthor(authorID int) []Review {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Review
	for _, r := range s.reviews {
		if r.AuthorID == authorID {
			result = append(result, r)
		}
	}
	return result
}

// FindByProduct returns all reviews for a specific product.
func (s *ReviewsStore) FindByProduct(product string) []Review {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Review
	for _, r := range s.reviews {
		if r.Product == product {
			result = append(result, r)
		}
	}
	return result
}

// Create adds a new review.
func (s *ReviewsStore) Create(body string, rating, authorID int, product string) Review {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := Review{ID: s.nextID, Body: body, Rating: rating, AuthorID: authorID, Product: product}
	s.nextID++
	s.reviews = append(s.reviews, r)
	return r
}

// --- Federated Response Types ---

// UserWithReviews extends User with reviews resolved from the reviews service.
type UserWithReviews struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Username string   `json:"username"`
	Reviews  []Review `json:"reviews"`
}

// ReviewWithAuthor extends Review with the author resolved from the users service.
type ReviewWithAuthor struct {
	ID      int    `json:"id"`
	Body    string `json:"body"`
	Rating  int    `json:"rating"`
	Product string `json:"product"`
	Author  *User  `json:"author"`
}

// --- Engine Setup ---

// setupUsersResolvers registers user-related query and mutation resolvers.
func setupUsersResolvers(engine *graphql.Engine, users *UsersStore, reviews *ReviewsStore) {
	// Query: users — returns all users with their reviews (federation join).
	engine.Query("users", func(ctx *graphql.ResolverContext) (any, error) {
		allUsers := users.FindAll()
		var result []UserWithReviews
		for _, u := range allUsers {
			uwr := UserWithReviews{
				ID:       u.ID,
				Name:     u.Name,
				Email:    u.Email,
				Username: u.Username,
				Reviews:  reviews.FindByAuthor(u.ID),
			}
			result = append(result, uwr)
		}
		return result, nil
	})

	// Query: user — returns a single user with their reviews by ID.
	engine.Query("user", func(ctx *graphql.ResolverContext) (any, error) {
		idVal, ok := ctx.Args["id"]
		if !ok {
			return nil, fmt.Errorf("argument 'id' is required")
		}
		id, ok := idVal.(float64)
		if !ok {
			return nil, fmt.Errorf("argument 'id' must be a number")
		}
		u := users.FindByID(int(id))
		if u == nil {
			return nil, fmt.Errorf("user #%d not found", int(id))
		}
		return UserWithReviews{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			Username: u.Username,
			Reviews:  reviews.FindByAuthor(u.ID),
		}, nil
	})

	// Mutation: createUser — creates a new user.
	engine.Mutation("createUser", func(ctx *graphql.ResolverContext) (any, error) {
		name, _ := ctx.Args["name"].(string)
		email, _ := ctx.Args["email"].(string)
		username, _ := ctx.Args["username"].(string)
		if name == "" || email == "" || username == "" {
			return nil, fmt.Errorf("name, email, and username are required")
		}
		u := users.Create(name, email, username)
		return UserWithReviews{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			Username: u.Username,
			Reviews:  nil,
		}, nil
	})
}

// setupReviewsResolvers registers review-related query and mutation resolvers.
func setupReviewsResolvers(engine *graphql.Engine, users *UsersStore, reviews *ReviewsStore) {
	// Query: reviews — returns all reviews with authors resolved from the users service.
	engine.Query("reviews", func(ctx *graphql.ResolverContext) (any, error) {
		allReviews := reviews.FindAll()
		var result []ReviewWithAuthor
		for _, r := range allReviews {
			rwa := ReviewWithAuthor{
				ID:      r.ID,
				Body:    r.Body,
				Rating:  r.Rating,
				Product: r.Product,
				Author:  users.FindByID(r.AuthorID),
			}
			result = append(result, rwa)
		}
		return result, nil
	})

	// Query: reviewsByProduct — returns reviews filtered by product name,
	// with authors resolved (cross-service reference).
	engine.Query("reviewsByProduct", func(ctx *graphql.ResolverContext) (any, error) {
		product, _ := ctx.Args["product"].(string)
		if product == "" {
			return nil, fmt.Errorf("argument 'product' is required")
		}
		productReviews := reviews.FindByProduct(product)
		var result []ReviewWithAuthor
		for _, r := range productReviews {
			rwa := ReviewWithAuthor{
				ID:      r.ID,
				Body:    r.Body,
				Rating:  r.Rating,
				Product: r.Product,
				Author:  users.FindByID(r.AuthorID),
			}
			result = append(result, rwa)
		}
		return result, nil
	})

	// Mutation: addReview — creates a review and resolves the author reference.
	engine.Mutation("addReview", func(ctx *graphql.ResolverContext) (any, error) {
		body, _ := ctx.Args["body"].(string)
		ratingVal, _ := ctx.Args["rating"].(float64)
		authorIDVal, _ := ctx.Args["authorId"].(float64)
		product, _ := ctx.Args["product"].(string)

		rating := int(ratingVal)
		authorID := int(authorIDVal)

		if body == "" || product == "" {
			return nil, fmt.Errorf("body and product are required")
		}
		if rating < 1 || rating > 5 {
			return nil, fmt.Errorf("rating must be between 1 and 5")
		}
		if users.FindByID(authorID) == nil {
			return nil, fmt.Errorf("author #%d not found", authorID)
		}

		r := reviews.Create(body, rating, authorID, product)
		return ReviewWithAuthor{
			ID:      r.ID,
			Body:    r.Body,
			Rating:  r.Rating,
			Product: r.Product,
			Author:  users.FindByID(r.AuthorID),
		}, nil
	})
}

// --- Module ---

// The federated schema (for documentation; the engine uses resolver names).
const federatedSchema = `
type User {
  id: Int!
  name: String!
  email: String!
  username: String!
  reviews: [Review!]!    # resolved from the reviews service
}

type Review {
  id: Int!
  body: String!
  rating: Int!
  product: String!
  author: User           # resolved from the users service (federation reference)
}

type Query {
  users: [User!]!
  user(id: Int!): User
  reviews: [Review!]!
  reviewsByProduct(product: String!): [Review!]!
}

type Mutation {
  createUser(name: String!, email: String!, username: String!): User!
  addReview(body: String!, rating: Int!, authorId: Int!, product: String!): Review!
}
`

func main() {
	// Create shared data stores (in a real federation these would be separate services).
	usersStore := NewUsersStore()
	reviewsStore := NewReviewsStore()

	// Build the GraphQL engine with resolvers from both "services".
	engine := graphql.NewEngine()
	setupUsersResolvers(engine, usersStore, reviewsStore)
	setupReviewsResolvers(engine, usersStore, reviewsStore)

	// Create the GraphQL module.
	gqlModule := graphql.NewModule(graphql.Options{
		Path:       "/graphql",
		Playground: true,
		Schema:     federatedSchema,
	}, engine)

	// Root application module.
	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{gqlModule},
	})

	// Bootstrap the application.
	app := gonest.Create(appModule)
	app.EnableCors()

	log.Println("GraphQL Federation example running at http://localhost:3000/graphql")
	log.Println("Playground available at http://localhost:3000/graphql (GET)")
	log.Println("")
	log.Println("Example queries:")
	log.Println("  { users }                              — all users with reviews")
	log.Println("  { reviews }                            — all reviews with authors")
	log.Println("  { reviewsByProduct(product: \"GoNest Framework\") } — reviews by product")
	log.Println("")
	log.Println("Example mutations:")
	log.Println("  mutation { createUser(name: \"Dave\", email: \"dave@example.com\", username: \"dave\") }")
	log.Println("  mutation { addReview(body: \"Great!\", rating: 5, authorId: 1, product: \"GoNest\") }")
	log.Fatal(app.Listen(":3000"))
}
