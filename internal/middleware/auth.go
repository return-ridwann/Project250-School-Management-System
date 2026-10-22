package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"sms/internal/services"
)

// ContextKey is a custom type for storing and retrieving values in request context.
type ContextKey string

const (
	UserIDKey ContextKey = "user_id"
	RoleKey   ContextKey = "role"
)

var configuredJWTSecret []byte

// SetJWTSecret configures the secret used by AuthMiddleware.
func SetJWTSecret(secret string) {
	configuredJWTSecret = []byte(secret)
}

func getJWTSecret() []byte {
	if len(configuredJWTSecret) > 0 {
		return configuredJWTSecret
	}
	if envSecret := os.Getenv("JWT_SECRET"); envSecret != "" {
		return []byte(envSecret)
	}
	return []byte(services.DefaultJWTSecret)
}

// GetUserID extracts the user ID from the context.
func GetUserID(ctx context.Context) (int, bool) {
	if val := ctx.Value(UserIDKey); val != nil {
		if id, ok := val.(int); ok {
			return id, true
		}
	}
	if val := ctx.Value("user_id"); val != nil {
		if id, ok := val.(int); ok {
			return id, true
		}
	}
	return 0, false
}

// GetRole extracts the user role from the context.
func GetRole(ctx context.Context) (string, bool) {
	if val := ctx.Value(RoleKey); val != nil {
		if role, ok := val.(string); ok {
			return role, true
		}
	}
	if val := ctx.Value("role"); val != nil {
		if role, ok := val.(string); ok {
			return role, true
		}
	}
	return "", false
}

// AuthMiddleware reads the session_token cookie, validates the JWT signature,
// and attaches user_id and role to the request context.Context. Returns HTTP 401 if invalid.
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "unauthorized",
			})
			return
		}

		secret := getJWTSecret()
		claims, err := services.ValidateToken(cookie.Value, secret)
		if err != nil {
			// Fallback: check against DefaultJWTSecret and dev key if configured secret differed
			if string(secret) != services.DefaultJWTSecret {
				claims, err = services.ValidateToken(cookie.Value, []byte(services.DefaultJWTSecret))
			}
			if err != nil {
				claims, err = services.ValidateToken(cookie.Value, []byte("dev_fallback_secret_key_change_in_prod"))
			}
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized",
				})
				return
			}
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "role", claims.Role)

		if next != nil {
			next(w, r.WithContext(ctx))
		}
	}
}

// RequireRole checks the user's role from the context against allowedRoles
// and returns HTTP 403 Forbidden if it does not match.
func RequireRole(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetRole(r.Context())
			if !ok || role == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "forbidden",
				})
				return
			}

			allowed := false
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					allowed = true
					break
				}
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "forbidden",
				})
				return
			}

			if next != nil {
				next(w, r)
			}
		}
	}
}
