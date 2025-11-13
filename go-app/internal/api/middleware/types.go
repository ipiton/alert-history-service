package middleware

// Context keys for middleware data storage
type contextKey string

const (
	// RequestIDContextKey is the context key for request ID
	RequestIDContextKey contextKey = "request_id"

	// UserContextKey is the context key for authenticated user
	UserContextKey contextKey = "user"

	// StartTimeContextKey is the context key for request start time
	StartTimeContextKey contextKey = "start_time"
)

// HTTP headers
const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"

	// AuthorizationHeader is the header name for authorization
	AuthorizationHeader = "Authorization"

	// RateLimitHeader prefix for rate limit headers
	RateLimitLimitHeader     = "X-RateLimit-Limit"
	RateLimitRemainingHeader = "X-RateLimit-Remaining"
	RateLimitResetHeader     = "X-RateLimit-Reset"

	// Cache control headers
	CacheControlHeader = "Cache-Control"
	ETagHeader         = "ETag"
	IfNoneMatchHeader  = "If-None-Match"

	// API version header
	APIVersionHeader = "X-API-Version"
)

// User represents an authenticated user
type User struct {
	ID       string
	Username string
	Role     string // viewer, operator, admin
	APIKey   string
}

// Role hierarchy levels
const (
	RoleViewer   = "viewer"
	RoleOperator = "operator"
	RoleAdmin    = "admin"
)

// Role hierarchy map for permission checking
var roleHierarchy = map[string]int{
	RoleViewer:   1,
	RoleOperator: 2,
	RoleAdmin:    3,
}

// HasRequiredRole checks if user role meets the required role level
func HasRequiredRole(userRole, requiredRole string) bool {
	userLevel, userExists := roleHierarchy[userRole]
	requiredLevel, requiredExists := roleHierarchy[requiredRole]

	if !userExists || !requiredExists {
		return false
	}

	return userLevel >= requiredLevel
}
