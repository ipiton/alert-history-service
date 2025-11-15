package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AuthenticationMiddleware validates authentication based on configured auth type
func AuthenticationMiddleware(config *AuthConfig) Middleware {
	// Skip if auth not enabled
	if !config.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var authenticated bool
			var err error

			switch config.Type {
			case "api_key":
				authenticated, err = validateAPIKey(r, config.APIKey)
			case "hmac":
				authenticated, err = validateHMAC(r, config.JWTSecret)
			default:
				err = fmt.Errorf("unsupported auth type: %s", config.Type)
			}

			if err != nil || !authenticated {
				if config.Logger != nil {
					config.Logger.Warn("Authentication failed",
						"request_id", GetRequestID(r.Context()),
						"client_ip", extractClientIP(r),
						"auth_type", config.Type,
						"error", err,
					)
				}

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="webhook", type="%s"`, config.Type))
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"status":  "unauthorized",
					"message": "Authentication required",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateAPIKey validates X-API-Key header against expected key
func validateAPIKey(r *http.Request, expectedKey string) (bool, error) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		return false, errors.New("missing X-API-Key header")
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(expectedKey)) != 1 {
		return false, errors.New("invalid API key")
	}

	return true, nil
}

// validateHMAC validates HMAC signature in X-Webhook-Signature header
func validateHMAC(r *http.Request, secret string) (bool, error) {
	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		return false, errors.New("missing X-Webhook-Signature header")
	}

	// Read body for signature calculation
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read body: %w", err)
	}
	
	// Restore body for downstream handlers
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Calculate HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := fmt.Sprintf("sha256=%x", mac.Sum(nil))

	// Compare signatures (constant-time)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) != 1 {
		return false, errors.New("invalid HMAC signature")
	}

	return true, nil
}

