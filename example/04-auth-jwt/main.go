package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gonest"
)

// Simple JWT implementation for demonstration

var jwtSecret = []byte("super-secret-key")

type JWTPayload struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
}

func createToken(userID, role string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := JWTPayload{Sub: userID, Role: role, Exp: time.Now().Add(24 * time.Hour).Unix()}
	payloadBytes, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := sign(header + "." + payloadB64)
	return header + "." + payloadB64 + "." + sig
}

func verifyToken(token string) (*JWTPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, gonest.NewUnauthorizedException("invalid token")
	}
	expected := sign(parts[0] + "." + parts[1])
	if expected != parts[2] {
		return nil, gonest.NewUnauthorizedException("invalid signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, gonest.NewUnauthorizedException("invalid payload")
	}
	var payload JWTPayload
	json.Unmarshal(payloadBytes, &payload)
	if payload.Exp < time.Now().Unix() {
		return nil, gonest.NewUnauthorizedException("token expired")
	}
	return &payload, nil
}

func sign(data string) string {
	h := hmac.New(sha256.New, jwtSecret)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// --- Auth Guard ---

type JWTGuard struct{}

func (g *JWTGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
	auth := ctx.Header("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false, gonest.NewUnauthorizedException("missing bearer token")
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	payload, err := verifyToken(token)
	if err != nil {
		return false, err
	}
	ctx.Set("user", payload)
	return true, nil
}

// --- Controllers ---

type AuthController struct{}

func NewAuthController() *AuthController { return &AuthController{} }

func (c *AuthController) Register(r gonest.Router) {
	r.Prefix("/auth")

	// Login with username + password, returns JWT
	r.Post("/login", c.login)
}

func (c *AuthController) login(ctx gonest.Context) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := ctx.Bind(&body); err != nil {
		return err
	}
	// Simple auth check
	if body.Username == "admin" && body.Password == "password" {
		token := createToken("1", "admin")
		return ctx.JSON(http.StatusOK, map[string]string{"access_token": token})
	}
	return gonest.NewUnauthorizedException("invalid credentials")
}

type ProfileController struct{}

func NewProfileController() *ProfileController { return &ProfileController{} }

func (c *ProfileController) Register(r gonest.Router) {
	r.Prefix("/profile")
	r.UseGuards(&JWTGuard{})

	// Get current user's profile (requires JWT)
	r.Get("/", c.getProfile)
}

func (c *ProfileController) getProfile(ctx gonest.Context) error {
	user, _ := ctx.Get("user")
	payload := user.(*JWTPayload)
	return ctx.JSON(http.StatusOK, map[string]string{
		"id":   payload.Sub,
		"role": payload.Role,
	})
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewAuthController, NewProfileController},
})

func main() {
	app := gonest.Create(AppModule)
	app.EnableCors()
	log.Fatal(app.Listen(":3000"))
}
