/*
The structure and content of the JWT is governed by sso.redhat.com, for more information
see https://source.redhat.com/groups/public/ciams/docs/external_sso_ssoredhatcom_claims__attributes
*/
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"
)

const (
	ScopeOfflineAccess = "offline_access"
)

var (
	ErrUnauthorizedScopes    = fmt.Errorf("token contains unauthorized scopes")
	ErrMissingRequiredScopes = fmt.Errorf("token is missing required scopes")
	ErrInvalidAudience       = fmt.Errorf("token audience does not match any approved audience")
)

// The default list of approved audiences that should be allowed to access OCM resource servers
var DefaultApprovedAudiences = []string{
	"cloud-services",
	"ocm-cli",
}

type TokenValidationMiddleware interface {
	Handler(next http.Handler) http.Handler
	ValidateAudience(ctx context.Context) error
	ValidateScopes(ctx context.Context) error
	ValidateAll(ctx context.Context) error
}

type TokenValidationMiddlewareImpl struct {
	approvedAudiences           []string
	denyScopes                  []string
	requiredScopes              []string
	enforceServiceAccountScopes bool
	callbackFn                  func(http.ResponseWriter, *http.Request, error)
}

var _ TokenValidationMiddleware = &TokenValidationMiddlewareImpl{}

func NewTokenValidationMiddleware(approvedAudiences []string, denyScopes []string, requiredScopes []string) *TokenValidationMiddlewareImpl {
	middleware := TokenValidationMiddlewareImpl{
		approvedAudiences: approvedAudiences,
		denyScopes:        denyScopes,
		requiredScopes:    requiredScopes,
	}

	return &middleware
}

// Validates the token aud claim contains at least one of the approved audience values.
// If approvedAudiences is empty then no validation is performed.
func (t *TokenValidationMiddlewareImpl) ValidateAudience(ctx context.Context) error {
	hasApprovedAudiences := len(t.approvedAudiences) > 0

	if !hasApprovedAudiences {
		return nil
	}

	claims, err := tokenClaimsFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token claims: %w", err)
	}

	if isServiceAccount(claims) {
		// service accounts do not have aud
		return nil
	}

	// aud can be a string or a slice of strings
	// if it's a string, convert it to a slice of strings
	aud, ok := claims["aud"].([]string)
	if !ok {
		audString, ok := claims["aud"].(string)
		if !ok {
			return fmt.Errorf("failed to get token aud")
		}
		aud = []string{audString}
	}

	validAudience := false
	for _, approvedAudience := range t.approvedAudiences {
		for _, audience := range aud {
			if approvedAudience == audience {
				validAudience = true
				break
			}
		}
		if validAudience {
			break
		}
	}

	if !validAudience {
		return ErrInvalidAudience
	}

	return nil
}

// Validates if the token scopes conform to the resource servers requirements.
// If denyScopes and requiredScopes are empty then no validation is performed.
func (t *TokenValidationMiddlewareImpl) ValidateScopes(ctx context.Context) error {
	hasDenyScopes := len(t.denyScopes) > 0
	hasRequiredScopes := len(t.requiredScopes) > 0

	if !hasDenyScopes && !hasRequiredScopes {
		// nothing to validate
		return nil
	}

	claims, err := tokenClaimsFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token claims: %w", err)
	}

	if isServiceAccount(claims) && !t.enforceServiceAccountScopes {
		return nil
	}

	scopes, ok := claims["scope"].(string)
	if !ok {
		return fmt.Errorf("failed to get token scopes")
	}
	scopesSplit := strings.Split(scopes, " ")

	// Get the token from context
	if hasRequiredScopes {
		missingScopes := []string{}
		for _, requiredScope := range t.requiredScopes {
			found := false
			for _, scope := range scopesSplit {
				if requiredScope == scope {
					found = true
					break
				}
			}
			if !found {
				missingScopes = append(missingScopes, requiredScope)
			}
		}
		if len(missingScopes) > 0 {
			return fmt.Errorf("%w: %v", ErrMissingRequiredScopes, missingScopes)
		}

	}
	if hasDenyScopes {
		deniedScopes := []string{}
		for _, denyScope := range t.denyScopes {
			for _, scope := range scopesSplit {
				if denyScope == scope {
					deniedScopes = append(deniedScopes, denyScope)
					break
				}
			}
		}
		if len(deniedScopes) > 0 {
			return fmt.Errorf("%w: %v", ErrUnauthorizedScopes, deniedScopes)
		}
	}
	return nil
}

// Validates the token context given the middleware configuration
func (t *TokenValidationMiddlewareImpl) ValidateAll(ctx context.Context) error {
	err := t.ValidateAudience(ctx)
	if err != nil {
		return err
	}
	err = t.ValidateScopes(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Runs ValidateAll and calls the next handler.
// Leverages the optional callbackFn for custom logging or error handling.
func (t *TokenValidationMiddlewareImpl) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := t.ValidateAll(r.Context())
		if t.callbackFn != nil {
			t.callbackFn(w, r, err)
		}
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extracts the JSON web token of the user from the context. If no token is found
// in the context then the result will be nil.
func tokenClaimsFromContext(ctx context.Context) (result jwt.MapClaims, err error) {
	tokenKeyValue := "token"
	switch token := ctx.Value(tokenKeyValue).(type) {
	case nil:
	case *jwt.Token:
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			err = fmt.Errorf("cannot convert token to claims")
		} else {
			result = claims
		}
	default:
		err = fmt.Errorf(
			"expected a token in the '%s' context value, but got '%T'",
			tokenKeyValue, token,
		)
	}

	return
}

func isServiceAccount(claims jwt.MapClaims) bool {
	_, clientIDExists := claims["client_id"]
	if !clientIDExists {
		_, clientIDExists = claims["clientId"]
	}
	return clientIDExists
}
