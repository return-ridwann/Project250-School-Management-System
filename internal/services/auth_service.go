package services

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"sms/internal/models"
	"sms/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTokenInvalid        = errors.New("invalid or expired token")
)

const DefaultJWTSecret = "sms-default-jwt-secret-key-2026"

// Claims defines JWT payload with user_id, role, and expiration.
type Claims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthService handles authentication and token management.
type AuthService struct {
	repo      *repository.Authrepo
	jwtSecret []byte
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *repository.Authrepo, secret string) *AuthService {
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = DefaultJWTSecret
		}
	}
	return &AuthService{
		repo:      repo,
		jwtSecret: []byte(secret),
	}
}

// VerifyPassword verifies a plaintext password against a bcrypt hash.
func VerifyPassword(plaintext, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
}

// CheckPasswordHash returns true if plaintext matches hash.
func CheckPasswordHash(plaintext, hash string) bool {
	return VerifyPassword(plaintext, hash) == nil
}

// GenerateToken generates a signed HS256 JWT token with user_id, role, and 24-hour expiration.
func GenerateToken(userID int, role string, secret []byte) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// GenerateToken generates an HS256 JWT using the AuthService secret.
func (s *AuthService) GenerateToken(userID int, role string) (string, error) {
	return GenerateToken(userID, role, s.jwtSecret)
}

// ValidateToken parses and verifies an HS256 JWT token.
func ValidateToken(tokenString string, secret []byte) (*Claims, error) {
	parsedToken, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, ErrTokenInvalid
	}

	claims, ok := parsedToken.Claims.(*Claims)
	if !ok || !parsedToken.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ValidateToken validates a token string against the AuthService secret.
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	return ValidateToken(tokenString, s.jwtSecret)
}

// Login validates credentials and returns a JWT token and user record.
func (s *AuthService) Login(email, password string) (string, *models.User, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := VerifyPassword(password, user.PasswordHash); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}
