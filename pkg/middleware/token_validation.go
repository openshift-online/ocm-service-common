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
	"github.com/openshift-online/ocm-sdk-go/authentication"
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
//   - cloud-services: Default UI & offline token client
//   - ocm-cli: OCM & ROSA CLI Authorizations
//   - customer-portal: Support case management use cases
//   - console-dot: Default FedRAMP keycloak client
var DefaultApprovedAudiences = []string{
	"cloud-services",
	"ocm-cli",
	"customer-portal",
	"console-dot",
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
	EnforceServiceAccountScopes bool
	CallbackFn                  func(http.ResponseWriter, *http.Request, error)
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

	for _, approvedAudience := range t.approvedAudiences {
		if claims.VerifyAudience(approvedAudience, true) {
			return nil
		}
	}
	return ErrInvalidAudience
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

	if isServiceAccount(claims) && !t.EnforceServiceAccountScopes {
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
		if t.CallbackFn != nil {
			t.CallbackFn(w, r, err)
		}
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extracts the JSON web token of the user from the context. If no token is found
// in the context then the result will be nil.
func tokenClaimsFromContext(ctx context.Context) (result jwt.MapClaims, err error) {
	token, err := authentication.TokenFromContext(ctx)
	if err != nil || token == nil {
		err = fmt.Errorf("cannot get token from context")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("cannot convert token to claims")
	} else {
		result = claims
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
