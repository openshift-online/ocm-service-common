package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/openshift-online/ocm-common/pkg/deprecation"
	"github.com/openshift-online/ocm-common/pkg/ocm/consts"
	"github.com/openshift-online/ocm-service-common/pkg/error"
)

// DeprecatedEndpoint represents a deprecated API endpoint with its message and sunset date.
type DeprecatedEndpoint struct {
	Message    string
	SunsetDate time.Time
}

type MiddlewareConfig struct {
	Endpoints              map[string]DeprecatedEndpoint
	CreateError            error.ErrorFactory
	SendError              error.SendErrorFunc
	EnableFieldDeprecation bool
}

// NewDeprecationMiddleware creates an HTTP middleware that adds deprecation headers
// and returns errors for expired endpoints. It accepts a map where keys are URL
// patterns and values are the deprecation details.
func NewDeprecationMiddleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the current request matches any deprecated endpoint
			deprecatedEndpoint, isDeprecated := matchDeprecatedEndpoint(r.URL.Path, cfg.Endpoints)
			if isDeprecated {
				now := time.Now().UTC()
				// Check if the endpoint is expired (sunset date is in the past)
				if now.After(deprecatedEndpoint.SunsetDate) {
					// Return a standard 410 Gone error for expired endpoints.
					body := cfg.CreateError(r, "%v", deprecatedEndpoint.Message)
					cfg.SendError(w, r, &body)
					return

				}
				// Add deprecation headers for active but deprecated endpoints
				w.Header().Set(consts.DeprecationHeader, deprecatedEndpoint.SunsetDate.Format(time.RFC3339))
				w.Header().Set(consts.OcmDeprecationMessage, deprecatedEndpoint.Message)
			}

			if cfg.EnableFieldDeprecation {
				ctx := r.Context()
				ctx = deprecation.WithFieldDeprecations(ctx)

				wrappedWriter := &deprecation.FieldDeprecationResponseWriter{
					ResponseWriter: w,
					Request:        r.WithContext(ctx),
				}

				next.ServeHTTP(wrappedWriter, r.WithContext(ctx))
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// matchDeprecatedEndpoint checks if the given path matches any deprecated endpoint
func matchDeprecatedEndpoint(path string, deprecatedEndpoints map[string]DeprecatedEndpoint) (DeprecatedEndpoint, bool) {
	// Direct match first
	if endpoint, exists := deprecatedEndpoints[path]; exists {
		return endpoint, true
	}
	// Pattern matching for endpoints with path parameters
	for pattern, endpoint := range deprecatedEndpoints {
		if matchesPattern(path, pattern) {
			return endpoint, true
		}
	}
	return DeprecatedEndpoint{}, false
}

// matchesPattern checks if a path matches a pattern with path parameters
func matchesPattern(path, pattern string) bool {
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(pathParts) != len(patternParts) {
		return false
	}
	for i, patternPart := range patternParts {
		// Skip path parameters (enclosed in curly braces)
		if strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}") {
			continue
		}
		// Exact match required for non-parameter parts
		if pathParts[i] != patternPart {
			return false
		}
	}
	return true
}
