package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gonest"
	"github.com/gonest/example/fullstack-api/common"
	"github.com/gonest/example/fullstack-api/users"
)

// AuthService handles authentication and token management.
type AuthService struct {
	usersService *users.UsersService
	secret       []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

func NewAuthService(usersService *users.UsersService) *AuthService {
	return &AuthService{
		usersService: usersService,
		secret:       []byte("change-me-in-production-use-env-var"),
		accessTTL:    15 * time.Minute,
		refreshTTL:   7 * 24 * time.Hour,
	}
}

func (s *AuthService) Register(dto RegisterDto) (*TokenResponse, error) {
	user, err := s.usersService.Create(users.CreateUserDto{
		Email:     dto.Email,
		Password:  dto.Password,
		FirstName: dto.FirstName,
		LastName:  dto.LastName,
		Role:      common.RoleUser,
	})
	if err != nil {
		return nil, err
	}
	return s.generateTokens(user)
}

func (s *AuthService) Login(dto LoginDto) (*TokenResponse, error) {
	user := s.usersService.FindByEmail(dto.Email)
	if user == nil {
		return nil, gonest.NewUnauthorizedException("invalid email or password")
	}
	if !s.usersService.VerifyPassword(user, dto.Password) {
		return nil, gonest.NewUnauthorizedException("invalid email or password")
	}
	if user.Status != "active" {
		return nil, gonest.NewForbiddenException("account is inactive")
	}
	return s.generateTokens(user)
}

func (s *AuthService) RefreshTokens(refreshToken string) (*TokenResponse, error) {
	payload, err := s.verifyToken(refreshToken)
	if err != nil {
		return nil, err
	}
	if payload.Type != "refresh" {
		return nil, gonest.NewUnauthorizedException("invalid token type")
	}
	user := s.usersService.FindByID(payload.Sub)
	if user == nil {
		return nil, gonest.NewUnauthorizedException("user not found")
	}
	return s.generateTokens(user)
}

func (s *AuthService) ValidateAccessToken(token string) (*common.AuthUser, error) {
	payload, err := s.verifyToken(token)
	if err != nil {
		return nil, err
	}
	if payload.Type != "access" {
		return nil, gonest.NewUnauthorizedException("invalid token type")
	}
	return &common.AuthUser{
		ID:    payload.Sub,
		Email: payload.Email,
		Role:  payload.Role,
	}, nil
}

// --- JWT implementation ---

type jwtPayload struct {
	Sub   int         `json:"sub"`
	Email string      `json:"email"`
	Role  common.Role `json:"role"`
	Type  string      `json:"type"` // "access" or "refresh"
	Exp   int64       `json:"exp"`
	Iat   int64       `json:"iat"`
}

func (s *AuthService) generateTokens(user *users.User) (*TokenResponse, error) {
	now := time.Now()

	accessPayload := jwtPayload{
		Sub:   user.ID,
		Email: user.Email,
		Role:  user.Role,
		Type:  "access",
		Iat:   now.Unix(),
		Exp:   now.Add(s.accessTTL).Unix(),
	}
	accessToken, err := s.signToken(accessPayload)
	if err != nil {
		return nil, gonest.NewInternalServerError("failed to generate access token")
	}

	refreshPayload := jwtPayload{
		Sub:  user.ID,
		Type: "refresh",
		Iat:  now.Unix(),
		Exp:  now.Add(s.refreshTTL).Unix(),
	}
	refreshToken, err := s.signToken(refreshPayload)
	if err != nil {
		return nil, gonest.NewInternalServerError("failed to generate refresh token")
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.accessTTL.Seconds()),
	}, nil
}

func (s *AuthService) signToken(payload jwtPayload) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := s.sign(header + "." + payloadB64)
	return header + "." + payloadB64 + "." + sig, nil
}

func (s *AuthService) verifyToken(token string) (*jwtPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, gonest.NewUnauthorizedException("malformed token")
	}

	expected := s.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, gonest.NewUnauthorizedException("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, gonest.NewUnauthorizedException("invalid token payload")
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, gonest.NewUnauthorizedException("invalid token payload")
	}

	if time.Now().Unix() > payload.Exp {
		return nil, gonest.NewUnauthorizedException("token expired")
	}

	return &payload, nil
}

func (s *AuthService) sign(data string) string {
	h := hmac.New(sha256.New, s.secret)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
