package middleware

import (
	"encoding/json"
	"errors"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"
)

const (
	OCMAccessScope = "api.ocm"
	OCMAudience    = "ocm-services"
)

type AccessRules struct {
	Roles []string `json:"roles"`
}

type OrganizationInfo struct {
	ID            *string `json:"id"`
	AccountNumber *string `json:"account_number"`
	Name          *string `json:"name"` // FedRamp: Keycloak-only
}

type OCMStandardClaims struct {
	Audience         *string          `json:"aud"`
	Subject          *string          `json:"sub"`
	IdentityProvider *string          `json:"idp"`
	Issuer           *string          `json:"iss"`
	Locale           *string          `json:"locale"`
	Scope            *string          `json:"scope"`
	ClientID         *string          `json:"client_id"`
	Email            *string          `json:"email"`
	EmailVerified    bool             `json:"email_verified"`
	Username         *string          `json:"preferred_username"`
	Organization     OrganizationInfo `json:"organization"`
	FirstName        *string          `json:"given_name"`
	LastName         *string          `json:"family_name"`
	RHITUserID       *string          `json:"user_id"`
	RHCreatorID      *string          `json:"rh-user-id"` // Org Service Account Creator User ID
	Access           AccessRules      `json:"realm_access"`
	Impersonated     bool             `json:"impersonated"`
	CognitoUsername  *string          `json:"username"`       // FedRamp: Cognito-only (non-oidc-standard: use preferred_username in all other cases)
	Groups           []string         `json:"cognito:groups"` // FedRamp: Cognito-only
}

// Commercial-only: Validates the scope/audience claims
func (a *OCMStandardClaims) CommercialIsValid() (bool, error) {
	if !strings.Contains(*a.Scope, OCMAccessScope) {
		return false, errors.New("invalid scope")
	}
	if !strings.Contains(*a.Audience, OCMAudience) {
		return false, errors.New("invalid audience")
	}
	return true, nil
}

func (a *OCMStandardClaims) UnmarshalFromToken(token *jwt.Token) error {
	raw, err := json.Marshal(token.Claims)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &a)
}

func JWTContainsOCMAccessScope(claims jwt.MapClaims) bool {
	if strings.Contains(claims["scope"].(string), OCMAccessScope) {
		return true
	}
	return false
}
