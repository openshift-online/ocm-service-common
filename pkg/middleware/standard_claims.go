package middleware

import (
	"encoding/json"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"
)

const (
	OCMIdentifier = "api.ocm"
)

type AccessRules struct {
	Roles []string `json:"roles"`
}

type OrganizationInfo struct {
	ID            *string `json:"id"`
	AccountNumber *string `json:"account_number"`
	Name          *string `json:"name"` // FedRamp: Keycloak-only
}

type AudienceRaw interface{}

// These standard claims are driven by the api.ocm access scope.
// The api.ocm access scope is active on the cloud-services client and can be manually enabled by SSO on central service accounts.
// Organization service accounts are enabled by default to use our custom scope and no enablement is required.
// The only exception here is Cognito authentication, as Cognito does not leverage our custom scopes but is compatible with these claims.
type OCMStandardClaims struct {
	Audience         []string
	AudienceRaw      AudienceRaw      `json:"aud,omitempty"` // aud can be a string "foo" or an array of strings ["foo", "bar"]
	Subject          *string          `json:"sub"`
	IdentityProvider *string          `json:"idp"` // Only present on login from Red Hat SSO
	Issuer           *string          `json:"iss"`
	Locale           *string          `json:"locale"`
	Scope            *string          `json:"scope"`
	ClientID         *string          `json:"clientId"` // Alternatively mapped from "client_id" for FedRAMP
	Email            *string          `json:"email"`
	EmailVerified    bool             `json:"email_verified"`
	Username         *string          `json:"preferred_username"`
	Organization     OrganizationInfo `json:"organization"`
	FirstName        *string          `json:"given_name"`
	LastName         *string          `json:"family_name"`
	RHITUserID       *string          `json:"user_id"`
	Access           AccessRules      `json:"realm_access"`
	Impersonated     bool             `json:"impersonated"`
	CognitoUsername  *string          `json:"username"`       // FedRAMP: Cognito-only normal user accounts (non-oidc-standard: use preferred_username in all other cases)
	Groups           []string         `json:"cognito:groups"` // FedRAMP: Cognito-only
	IsOrgAdmin       bool             `json:"is_org_admin"`   // FedRAMP: Keycloak-only

	// Organizational Service Accounts-only
	RHCreatorID *string `json:"rh-user-id"` // Org Service Account Creator User ID
}

func (a *OCMStandardClaims) UnmarshalJSON(b []byte) error {

	type TmpClaims OCMStandardClaims
	var tmpClaims TmpClaims

	err := json.Unmarshal(b, &tmpClaims)
	if err != nil {
		return err
	}

	// Map AudienceRaw to Audience based on type
	switch aud := tmpClaims.AudienceRaw.(type) {
	case string:
		tmpClaims.Audience = []string{aud}
	case []interface{}:
		tmpClaims.Audience = make([]string, len(aud))
		for i, v := range aud {
			tmpClaims.Audience[i] = v.(string)
		}
	default:
		tmpClaims.Audience = nil
	}

	*a = OCMStandardClaims(tmpClaims)

	return nil
}

func (a *OCMStandardClaims) UnmarshalFromToken(token *jwt.Token) error {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return jwt.NewValidationError("cannot convert claims", jwt.ValidationErrorClaimsInvalid)
	}

	// Fallback to mapping client_id to clientId
	// This currently differs between commercial (clientId) and FedRAMP (client_id)
	if claims["clientId"] == nil && claims["client_id"] != nil {
		claims["clientId"] = claims["client_id"]
	}

	raw, err := json.Marshal(claims)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &a)
}

// VerifyOCMClaims compares Scope and Audience claims to ensure they contain "api.ocm", this should be called before
// mapping claims to the OCMStandardClaims struct.
func VerifyOCMClaims(claims jwt.MapClaims) bool {
	iss, issExists := claims["iss"]
	_, audExists := claims["aud"]
	scope, scopeExists := claims["scope"]
	clientID, clientIDExists := claims["clientId"]

	isCognito := issExists && iss != nil && strings.Contains(iss.(string), "cognito")

	// Cognito does not support custom scopes or claims - return verified by default
	if isCognito {
		return true
	}

	if !scopeExists {
		return false
	}

	if clientIDExists && clientID != nil {
		// If client_id is present this is a service account which could contain a hard-coded custom audience
		// Thus, we only care about validating the scope
		return strings.Contains(scope.(string), OCMIdentifier)
	}

	if !audExists {
		return false
	}

	return strings.Contains(scope.(string), OCMIdentifier) &&
		claims.VerifyAudience(OCMIdentifier, true)
}
