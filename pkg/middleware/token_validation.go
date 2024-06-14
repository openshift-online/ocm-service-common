/*
The structure and content of the JWTs we are validating is governed by sso.redhat.com, for more information
see https://source.redhat.com/groups/public/ciams/docs/external_sso_ssoredhatcom_claims__attributes
*/
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	authv1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/pkg/errors"
)

const (
	// Feature flag that enables enforcing offline token restriction
	FlagEnforceOfflineTokenRestrictions = "enforce-offline-token-restrictions"
	// Controls an organizations ability to use offline tokens, if the above flag is enabled
	OfflineAccessCapabilityKey = "capability.organization.restrict_offline_tokens"
	ScopeOfflineAccess         = "offline_access"
	ClaimOrgId                 = "org_id"
	ClaimScope                 = "scope"
	ClaimOrganization          = "organization"
	ClaimId                    = "id"
	ClaimClientId              = "client_id"
	ClaimClientIdLegacy        = "clientId"
)

var (
	ErrUnauthorizedScopes    = fmt.Errorf("token contains unauthorized scopes")
	ErrMissingRequiredScopes = fmt.Errorf("token is missing required scopes")
	ErrMissingToken          = fmt.Errorf("missing token in context")
	ErrMissingSDKConnection  = fmt.Errorf("OCM SDK connection is missing")
)

type TokenScopeValidationMiddleware interface {
	Handler(next http.Handler) http.Handler
	ValidateAll(ctx context.Context) error
	ValidateScopes(ctx context.Context) error
	ValidateOfflineAccessByOrg(ctx context.Context) error

	StartPollingAMSForRestrictedOrgs() context.CancelFunc
	Start(ctx context.Context, ulog logging.Logger)
}

// TokenScopeValidationMiddlewareImpl provides a middleware that enables validation on the JWT token of an incoming request
// based on the configuration provided.
//
// Types of validation provided:
//   - Scopes: Can validate that the token scopes conform to the resource server's requirements.
//   - Offline Token Access: Can validate offline token access for an organization in the token context.
//
// This middleware can be safely placed on your root router as long as ErrorOnMissingToken is not set to true. If it is set
// to true, you run the risk of returning an error on any unauthenticated endpoints that should not require a token. In
// that case you should place this middleware only on authenticated routes."
// Configuration for the token validation middleware
//   - Connection: The OCM SDK connection to use for the middleware.
//   - DisableAllValidation: If true, the middleware will not perform any validation. Provides an escape hatch for disabling/enabling the middleware.
//   - ErrorOnMissingToken: If true, the middleware will return an error if it receives a request without a token.
//     This should NOT be true if you are appending this middleware to your top-level router.
//   - DenyScopes: A list of scope values that are not allowed to access the resource server. Such as `offline_access`.
//   - RequiredScopes: A list of scope values that are required to access the resource server. Such as `api.ocm`.
//   - CallbackFn: An optional function that can allow for custom logging or error handling post-validation.
//     The middleware will always call this function if provided.
//   - EnforceServiceAccountScopes: If true, the middleware will enforce the required and deny scopes on service accounts.
//   - PollingIntervalOverride: Optional override for the default 15 minute polling interval for offline org restrictions.
type TokenScopeValidationMiddlewareImpl struct {
	mu                            sync.Mutex // safe "concurrent" map access
	offlineRestrictedOrgs         map[string]bool
	enforceOfflineOrgRestrictions bool // Enabled via feature flag
	Connection                    *sdk.Connection
	DisableAllValidation          bool
	ErrorOnMissingToken           bool
	DenyScopes                    []string
	RequiredScopes                []string
	CallbackFn                    func(http.ResponseWriter, *http.Request, error)
	EnforceServiceAccountScopes   bool
	PollingIntervalOverride       time.Duration
	Logger                        logging.Logger
}

var _ TokenScopeValidationMiddleware = &TokenScopeValidationMiddlewareImpl{}

// Runs ValidateAll and calls the next handler.
// Leverages the optional callbackFn for custom logging or error handling.
func (t *TokenScopeValidationMiddlewareImpl) Handler(next http.Handler) http.Handler {
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

// Validates the token context given the middleware configuration
func (t *TokenScopeValidationMiddlewareImpl) ValidateAll(ctx context.Context) error {
	if t.DisableAllValidation {
		return nil
	}

	err := t.ValidateOfflineAccessByOrg(ctx)
	if err != nil {
		return err
	}
	err = t.ValidateScopes(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Validates if the token scopes conform to the resource servers requirements.
// If denyScopes and requiredScopes are empty then no validation is performed.
func (t *TokenScopeValidationMiddlewareImpl) ValidateScopes(ctx context.Context) error {
	if t.DisableAllValidation {
		return nil
	}

	hasDenyScopes := len(t.DenyScopes) > 0
	hasRequiredScopes := len(t.RequiredScopes) > 0

	if !hasDenyScopes && !hasRequiredScopes {
		// nothing to validate
		return nil
	}

	claims, err := tokenClaimsFromContext(ctx)
	if !t.ErrorOnMissingToken && errors.Is(err, ErrMissingToken) {
		// We did not find a token, and it was not due to bad token context
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get token claims: %w", err)
	}

	if isServiceAccount(claims) && !t.EnforceServiceAccountScopes {
		return nil
	}

	scopes, ok := claims[ClaimScope].(string)
	if !ok {
		// If we don't find token scopes, there is nothing to validate
		return nil
	}
	scopesSplit := strings.Split(scopes, " ")

	// Get the token from context
	if hasRequiredScopes {
		missingScopes := []string{}
		for _, requiredScope := range t.RequiredScopes {
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
		for _, denyScope := range t.DenyScopes {
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

// Validates offline access for the organization in the token context
// Requires the OCM SDK connection to be set and StartPollingAMSForRestrictedOrgs to be called.
func (t *TokenScopeValidationMiddlewareImpl) ValidateOfflineAccessByOrg(ctx context.Context) error {
	if t.DisableAllValidation || !t.isOfflineOrgRestrictionsEnabledSafe() {
		return nil
	}

	// Validate there are organizations to restrict
	hasRestrictedOrgs := t.getOfflineRestrictedOrgCountSafe() > 0
	if !hasRestrictedOrgs {
		return nil
	}

	claims, err := tokenClaimsFromContext(ctx)
	if !t.ErrorOnMissingToken && errors.Is(err, ErrMissingToken) {
		// We did not find a token, and it was not due to an error
		return nil
	}
	if err != nil {
		return err
	}

	if isServiceAccount(claims) {
		// Service accounts do not have offline access
		return nil
	}

	scopes, ok := claims[ClaimScope].(string)
	if !ok {
		// If we don't find token scopes, there is nothing to validate
		return nil
	}

	if !strings.Contains(scopes, ScopeOfflineAccess) {
		// The token does not contain offline access, no enforcement needed
		return nil
	}

	// Grab organization ID from token, with fallback for the access scope claim
	orgID, ok := claims[ClaimOrgId].(string)
	if !ok {
		orgClaim := claims[ClaimOrganization]
		if orgClaim != nil {
			// Fallback for access scope claim
			orgClaimMap, ok := orgClaim.(map[string]interface{})
			if ok {
				orgID, ok = orgClaimMap[ClaimId].(string)
				if !ok {
					return fmt.Errorf("failed to get organization id from token")
				}
			}
		}
	}

	if t.isOrgRestrictedSafe(orgID) {
		return fmt.Errorf("offline access is restricted for organization %s", orgID)
	}

	return nil
}

// Immediately populates the offline restricted orgs & feature flag, then starts a polling
// function to repeat the operation. This should be called once at the start
// of the application, before the server starts accepting requests.
// For services that will let ocm-common manage the routine
func (t *TokenScopeValidationMiddlewareImpl) StartPollingAMSForRestrictedOrgs() context.CancelFunc {
	ulog := t.Logger
	if ulog == nil {
		ulog, _ = sdk.NewGoLoggerBuilder().
			Info(true).
			Build()
	}
	ctx, cancel := context.WithCancel(context.Background())

	if t.Connection == nil {
		ulog.Debug(ctx, "OCM SDK connection is missing, offline token restrictions will not be enforced")
		return cancel
	}
	t.preSteps(ctx, ulog)

	pollTicker := t.createTicker(ctx, ulog)
	go func() {
		for {
			select {
			case <-pollTicker.C:
				t.check(ctx, ulog)
			case <-ctx.Done():
				pollTicker.Stop()
				return
			}
		}
	}()

	return cancel
}

func (t *TokenScopeValidationMiddlewareImpl) createTicker(ctx context.Context, ulog logging.Logger) *time.Ticker {
	// Schedule the polling function to run every 5 minutes or the override duration
	duration := 5 * time.Minute
	if t.PollingIntervalOverride > 0 {
		duration = t.PollingIntervalOverride
	}

	ulog.Info(ctx, "Created polling ticker for AMS org restrictions every %v...", duration)
	return time.NewTicker(duration)
}

// For services that have a routine manager
func (t *TokenScopeValidationMiddlewareImpl) Start(ctx context.Context, ulog logging.Logger) {
	if t.Connection == nil {
		ulog.Debug(ctx, "OCM SDK connection is missing, offline token restrictions will not be enforced")
		return
	}
	t.preSteps(ctx, ulog)

	pollTicker := t.createTicker(ctx, ulog)
	for {
		select {
		case <-pollTicker.C:
			t.check(ctx, ulog)
		case <-ctx.Done():
			pollTicker.Stop()
			return
		}
	}
}

func (t *TokenScopeValidationMiddlewareImpl) preSteps(ctx context.Context, ulog logging.Logger) {
	successfulOrgInit, _ := t.populateOfflineRestrictedOrgs()

	if successfulOrgInit {
		ulog.Info(ctx, "Successfully initialized offline access org restrictions, org list: %v total orgs",
			t.getOfflineRestrictedOrgCountSafe())
	} else {
		// Log and continue, the goroutine will attempt to self-heal.
		// API requests will fail open and offline access will be allowed.
		ulog.Error(ctx,
			"Failed to initialize offline access restricted orgs, restrictions will not be applied until self-healing occurs",
		)
	}

	successfulFlagInit, _ := t.checkEnforceOfflineOrgRestrictions(ctx)

	enforceOfflineOrgRestrictions := t.isOfflineOrgRestrictionsEnabledSafe()

	if successfulFlagInit {
		ulog.Info(ctx, "Successfully initialized offline enforcement flag: %t", enforceOfflineOrgRestrictions)
	} else {
		// Log and continue, the goroutine will attempt to self-heal.
		// API requests will fail open and offline access will be allowed.
		ulog.Error(ctx,
			"Failed to initialize offline enforcement flag, restrictions will not be applied until self-healing occurs",
		)
	}
}

func (t *TokenScopeValidationMiddlewareImpl) check(
	ctx context.Context, ulog logging.Logger) {
	// Populate the orgs
	ulog.Info(ctx, "Polling AMS for org restrictions...")
	if _, err := t.populateOfflineRestrictedOrgs(); err != nil {
		ulog.Error(ctx, "Failed AMS polling for org restrictions: %v", err)
		ulog.Info(ctx, "Continuing to use existing org list: %d total orgs", t.getOfflineRestrictedOrgCountSafe())
	} else {
		ulog.Info(ctx, "Successfully populated offline access org restrictions, org list: %d total orgs",
			t.getOfflineRestrictedOrgCountSafe())
	}

	// Check the feature flag
	ulog.Info(ctx, "Checking feature flag for offline token enforcement...")
	if _, err := t.checkEnforceOfflineOrgRestrictions(ctx); err != nil {
		ulog.Error(ctx, "Failed to check feature flag for offline token enforcement: %v", err)
		ulog.Info(ctx, "Continuing to use existing flag value of %t", t.isOfflineOrgRestrictionsEnabledSafe())
	} else {
		ulog.Info(ctx, "Successfully populated feature flag for offline token enforcement, flag value: %t",
			t.isOfflineOrgRestrictionsEnabledSafe())
	}
}

// Checks the feature flag to enable offline org restrictions
// Returns true if the operation was successful, false otherwise
func (t *TokenScopeValidationMiddlewareImpl) checkEnforceOfflineOrgRestrictions(ctx context.Context) (bool, error) {
	isFlagEnabled, err := t.isFeatureEnabled(ctx, FlagEnforceOfflineTokenRestrictions)
	if errors.Is(err, ErrMissingSDKConnection) {
		// Do not retry if we are missing the SDK connection
		return true, nil
	}
	if err != nil {
		return false, err
	}
	// Set the flag to enable offline org restrictions
	t.setEnforceOfflineOrgRestrictionsSafe(isFlagEnabled)
	return true, nil
}

// Populates t.offlineRestrictedOrgs map with the result from AMS labels and organizations API
// Returns true if the operation was successful, false otherwise
func (t *TokenScopeValidationMiddlewareImpl) populateOfflineRestrictedOrgs() (bool, error) {
	offlineRestrictedOrgs := make(map[string]bool)

	if t.Connection == nil {
		// Do not retry if we are missing the SDK connection
		return true, nil
	}
	api := t.Connection.AccountsMgmt().V1()
	labelResponse, err := api.Labels().List().Search(
		fmt.Sprintf("key = '%s'", OfflineAccessCapabilityKey) +
			" and internal = true and value = 'true'",
	).Send()
	if err != nil {
		return false, err
	}

	if labelResponse == nil || labelResponse.Items() == nil ||
		len(labelResponse.Items().Slice()) == 0 {
		// No offline restricted orgs found, set the map to empty and return
		t.setOfflineRestrictedOrgsSafe(offlineRestrictedOrgs)
		return true, nil
	}

	organizations := []string{}
	labelResponse.Items().Each(func(item *v1.Label) bool {
		organizations = append(organizations, item.OrganizationID())
		return true
	})

	quotedOrganizations := make([]string, len(organizations))
	for i, org := range organizations {
		quotedOrganizations[i] = fmt.Sprintf("'%s'", org)
	}

	// Fetch external org IDs to match on the JWT claim
	orgResponse, err := api.Organizations().List().
		Search(fmt.Sprintf("id in (%s)", strings.Join(quotedOrganizations, ", "))).Send()
	if err != nil {
		// We have organizations to restrict, but failed to fetch them
		// Do not reset the map, we will retry on the next polling interval
		return false, err
	}
	orgResponse.Items().Each(func(item *v1.Organization) bool {
		externalId := item.ExternalID()
		offlineRestrictedOrgs[externalId] = true
		return true
	})

	t.setOfflineRestrictedOrgsSafe(offlineRestrictedOrgs)

	return true, err
}

func (t *TokenScopeValidationMiddlewareImpl) getOfflineRestrictedOrgCountSafe() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.offlineRestrictedOrgs)
}

func (t *TokenScopeValidationMiddlewareImpl) isOrgRestrictedSafe(orgID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.offlineRestrictedOrgs[orgID]
}

func (t *TokenScopeValidationMiddlewareImpl) setOfflineRestrictedOrgsSafe(offlineRestrictedOrgs map[string]bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.offlineRestrictedOrgs = offlineRestrictedOrgs
}

func (t *TokenScopeValidationMiddlewareImpl) isOfflineOrgRestrictionsEnabledSafe() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.enforceOfflineOrgRestrictions
}

func (t *TokenScopeValidationMiddlewareImpl) setEnforceOfflineOrgRestrictionsSafe(enforceOfflineOrgRestrictions bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enforceOfflineOrgRestrictions = enforceOfflineOrgRestrictions
}

func (t *TokenScopeValidationMiddlewareImpl) isFeatureEnabled(ctx context.Context, featureName string) (bool, error) {
	if t.Connection == nil {
		return false, ErrMissingSDKConnection
	}
	authorizations := t.Connection.Authorizations().V1()

	builder := authv1.FeatureReviewRequestBuilder{}
	request, err := builder.Feature(featureName).Build()
	if err != nil {
		return false, err
	}
	response, err := authorizations.FeatureReview().Post().Request(request).SendContext(ctx)
	if err != nil {
		return false, err
	}
	if response.Status() != http.StatusOK {
		return false, errors.Errorf("got http %d trying to query AMS for feature review of feature '%s'",
			response.Status(), featureName)
	}
	return response.Request().Enabled(), nil
}

// extracts the JSON web token of the user from the context. If no token is found
// in the context then the result will be nil.
func tokenClaimsFromContext(ctx context.Context) (result jwt.MapClaims, err error) {
	token, err := authentication.TokenFromContext(ctx)

	if err == nil && token == nil {
		return nil, ErrMissingToken
	}

	if err != nil || token == nil {
		return nil, fmt.Errorf("invalid token in context")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("cannot convert token to claims")
	} else {
		result = claims
	}

	return result, err
}

func isServiceAccount(claims jwt.MapClaims) bool {
	_, clientIDExists := claims[ClaimClientId]
	if !clientIDExists {
		_, clientIDExists = claims[ClaimClientIdLegacy]
	}
	return clientIDExists
}
