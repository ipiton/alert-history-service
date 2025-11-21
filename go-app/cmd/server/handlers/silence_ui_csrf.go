// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// CSRFManager manages CSRF tokens for form protection.
// Phase 12: Error Handling enhancement.
type CSRFManager struct {
	tokens    map[string]*csrfToken
	mu        sync.RWMutex
	secretKey []byte
	ttl       time.Duration
	logger    *slog.Logger
}

type csrfToken struct {
	token     string
	createdAt time.Time
	expiresAt time.Time
}

// NewCSRFManager creates a new CSRF manager.
func NewCSRFManager(secretKey []byte, ttl time.Duration, logger *slog.Logger) *CSRFManager {
	if len(secretKey) == 0 {
		// Generate a random secret key if not provided
		secretKey = make([]byte, 32)
		if _, err := rand.Read(secretKey); err != nil {
			logger.Warn("Failed to generate CSRF secret key, using default", "error", err)
		}
	}

	return &CSRFManager{
		tokens:    make(map[string]*csrfToken),
		secretKey: secretKey,
		ttl:       ttl,
		logger:    logger,
	}
}

// GenerateToken generates a new CSRF token for a session.
func (cm *CSRFManager) GenerateToken(sessionID string) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)
	now := time.Now()

	cm.mu.Lock()
	cm.tokens[sessionID] = &csrfToken{
		token:     token,
		createdAt: now,
		expiresAt: now.Add(cm.ttl),
	}
	cm.mu.Unlock()

	cm.logger.Debug("CSRF token generated", "session_id", sessionID)

	return token, nil
}

// ValidateToken validates a CSRF token for a session.
func (cm *CSRFManager) ValidateToken(sessionID, token string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stored, exists := cm.tokens[sessionID]
	if !exists {
		cm.logger.Warn("CSRF token not found for session", "session_id", sessionID)
		return false
	}

	// Check expiration
	if time.Now().After(stored.expiresAt) {
		cm.logger.Warn("CSRF token expired", "session_id", sessionID)
		return false
	}

	// Compare tokens
	if stored.token != token {
		cm.logger.Warn("CSRF token mismatch", "session_id", sessionID)
		return false
	}

	return true
}

// CleanupExpiredTokens removes expired tokens from the cache.
func (cm *CSRFManager) CleanupExpiredTokens() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	removed := 0

	for sessionID, token := range cm.tokens {
		if now.After(token.expiresAt) {
			delete(cm.tokens, sessionID)
			removed++
		}
	}

	if removed > 0 {
		cm.logger.Debug("Cleaned up expired CSRF tokens", "removed", removed)
	}
}

// StartCleanupWorker starts a background worker to clean up expired tokens.
func (cm *CSRFManager) StartCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.CleanupExpiredTokens()
		}
	}
}

// getSessionID extracts session ID from request (cookie or header).
func (h *SilenceUIHandler) getSessionID(r *http.Request) string {
	// Try to get from cookie
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Fallback to IP address + User-Agent hash
	ip := r.RemoteAddr
	ua := r.UserAgent()
	return fmt.Sprintf("%s:%s", ip, ua)
}

// generateCSRFToken generates a CSRF token for the current request.
func (h *SilenceUIHandler) generateCSRFToken(r *http.Request) string {
	if h.csrfManager == nil {
		// Fallback to placeholder if CSRF manager not initialized
		return "csrf-token-placeholder"
	}

	sessionID := h.getSessionID(r)
	token, err := h.csrfManager.GenerateToken(sessionID)
	if err != nil {
		h.logger.Warn("Failed to generate CSRF token", "error", err)
		return "csrf-token-placeholder"
	}

	return token
}

// validateCSRFToken validates a CSRF token from the request.
func (h *SilenceUIHandler) validateCSRFToken(r *http.Request) bool {
	if h.csrfManager == nil {
		// Skip validation if CSRF manager not initialized
		return true
	}

	// Get token from form or header
	token := r.FormValue("csrf_token")
	if token == "" {
		token = r.Header.Get("X-CSRF-Token")
	}

	if token == "" {
		h.logger.Warn("CSRF token missing from request")
		return false
	}

	sessionID := h.getSessionID(r)
	return h.csrfManager.ValidateToken(sessionID, token)
}
