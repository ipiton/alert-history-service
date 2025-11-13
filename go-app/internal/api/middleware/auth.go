package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	// API keys mapped to users
	// Key: API key, Value: User
	APIKeys map[string]*User

	// JWT secret for token validation (future)
	JWTSecret string

	// Enable API key authentication
	EnableAPIKey bool

	// Enable JWT authentication
	EnableJWT bool
}

// AuthMiddleware validates API key or JWT token
//
// Supported authentication types:
//   - ApiKey: Header "Authorization: ApiKey <key>"
//   - Bearer: Header "Authorization: Bearer <jwt>" (future)
//
// On success, adds User to request context (accessible via UserContextKey).
// On failure, returns 401 Unauthorized.
func AuthMiddleware(config AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get(AuthorizationHeader)
			if authHeader == "" {
				writeUnauthorized(w, r, "Missing Authorization header")
				return
			}

			// Parse authorization header
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				writeUnauthorized(w, r, "Invalid Authorization header format")
				return
			}

			authType := parts[0]
			authValue := parts[1]

			var user *User
			var err error

			switch authType {
			case "ApiKey":
				if !config.EnableAPIKey {
					writeUnauthorized(w, r, "API key authentication disabled")
					return
				}
				user, err = validateAPIKey(authValue, config.APIKeys)

			case "Bearer":
				if !config.EnableJWT {
					writeUnauthorized(w, r, "JWT authentication disabled")
					return
				}
				user, err = validateJWT(authValue, config.JWTSecret)

			default:
				writeUnauthorized(w, r, "Unsupported authentication type")
				return
			}

			if err != nil || user == nil {
				writeUnauthorized(w, r, "Invalid credentials")
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// validateAPIKey validates API key against configuration
func validateAPIKey(apiKey string, apiKeys map[string]*User) (*User, error) {
	if user, exists := apiKeys[apiKey]; exists {
		return user, nil
	}
	return nil, nil
}

// validateJWT validates JWT token (placeholder for future implementation)
func validateJWT(token string, secret string) (*User, error) {
	// TODO: Implement JWT validation
	// - Parse token
	// - Verify signature
	// - Check expiration
	// - Extract claims
	// - Return User
	return nil, nil
}

// RBACMiddleware checks if user has required role
//
// Role hierarchy: admin (3) > operator (2) > viewer (1)
//
// Returns 403 Forbidden if user lacks required permissions.
func RBACMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserContextKey).(*User)
			if !ok || user == nil {
				writeUnauthorized(w, r, "User not authenticated")
				return
			}

			if !HasRequiredRole(user.Role, requiredRole) {
				writeForbidden(w, r, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AdminMiddleware is a convenience wrapper for admin-only endpoints
func AdminMiddleware(next http.Handler) http.Handler {
	return RBACMiddleware(RoleAdmin)(next)
}

// OperatorMiddleware is a convenience wrapper for operator+ endpoints
func OperatorMiddleware(next http.Handler) http.Handler {
	return RBACMiddleware(RoleOperator)(next)
}

// writeUnauthorized writes 401 Unauthorized response
func writeUnauthorized(w http.ResponseWriter, r *http.Request, message string) {
	requestID := GetRequestID(r.Context())
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":       "AUTHENTICATION_ERROR",
			"message":    message,
			"request_id": requestID,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(errorResponse)
}

// writeForbidden writes 403 Forbidden response
func writeForbidden(w http.ResponseWriter, r *http.Request, message string) {
	requestID := GetRequestID(r.Context())
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":       "AUTHORIZATION_ERROR",
			"message":    message,
			"request_id": requestID,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(errorResponse)
}

// GetUser extracts authenticated user from context
func GetUser(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(UserContextKey).(*User)
	return user, ok
}
