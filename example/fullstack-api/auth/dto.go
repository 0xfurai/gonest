package auth

// RegisterDto is used for user registration.
type RegisterDto struct {
	Email     string `json:"email" validate:"required,email" swagger:"example=user@example.com"`
	Password  string `json:"password" validate:"required,min=8,max=128" swagger:"example=password123"`
	FirstName string `json:"firstName" validate:"required,min=1,max=100" swagger:"example=John"`
	LastName  string `json:"lastName" validate:"required,min=1,max=100" swagger:"example=Doe"`
}

// LoginDto is used for login.
type LoginDto struct {
	Email    string `json:"email" validate:"required,email" swagger:"example=admin@example.com"`
	Password string `json:"password" validate:"required" swagger:"example=admin123"`
}

// TokenResponse is returned on successful auth.
type TokenResponse struct {
	AccessToken  string `json:"accessToken" swagger:"example=eyJhbGciOiJIUzI1NiJ9..."`
	RefreshToken string `json:"refreshToken" swagger:"example=eyJhbGciOiJIUzI1NiJ9..."`
	ExpiresIn    int    `json:"expiresIn" swagger:"example=900"`
}

// RefreshDto is used to refresh an access token.
type RefreshDto struct {
	RefreshToken string `json:"refreshToken" validate:"required" swagger:"example=eyJhbGciOiJIUzI1NiJ9..."`
}
