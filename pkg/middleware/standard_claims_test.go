package middleware

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/golang-jwt/jwt/v4"
)

var CommercialValidClaims = jwt.MapClaims{
	"aud":                OCMIdentifier,
	"sub":                "foo",
	"idp":                "auth.redhat.com",
	"iss":                "https://sso.stage.redhat.com/auth/realms/redhat-external",
	"locale":             "en_US",
	"scope":              fmt.Sprintf("openid %s", OCMIdentifier),
	"email":              "foo@bar.com",
	"email_verified":     true,
	"preferred_username": "foo",
	"organization": map[string]interface{}{
		"id":             "foo",
		"account_number": "foo",
	},
	"given_name":  "foo",
	"family_name": "bar",
	"user_id":     "foo",
	"rh-user-id":  "foo",
	"realm_access": map[string]interface{}{
		"roles": []string{
			"foo",
			"bar",
		},
	},
}
var CommercialServiceAccountValidClaims = jwt.MapClaims{
	"aud":                OCMIdentifier,
	"sub":                "foo",
	"iss":                "https://sso.stage.redhat.com/auth/realms/redhat-external",
	"locale":             "en_US",
	"scope":              fmt.Sprintf("openid %s", OCMIdentifier),
	"clientId":           "service-account-foo",
	"preferred_username": "service-account-foo",
}
var CommercialOrgServiceAccountValidClaims = jwt.MapClaims{
	"aud":            OCMIdentifier,
	"sub":            "foo-bar-uuid",
	"iss":            "https://sso.stage.redhat.com/auth/realms/redhat-external",
	"scope":          OCMIdentifier,
	"email_verified": false,
	"clientId":       "service-account-foo",
	"rh-user-id":     "12345678",
	"organization": map[string]interface{}{
		"id": "12345678",
	},
	"preferred_username": "service-account-foo",
}
var CognitoFedRampValidClaims = jwt.MapClaims{
	"sub":       "foo",
	"iss":       "https://cognito-idp.foobar.amazonaws.com/foobar",
	"scope":     "openid",
	"username":  "foo",
	"client_id": "bar", // included from cognito for SA and non-SA accounts
	"cognito:groups": []string{
		"orgName",
	},
}
var KeycloakFedRampValidClaims = jwt.MapClaims{
	"sub":                "foo",
	"iss":                "https://foobar.com/foobar",
	"scope":              "openid api.ocm",
	"email":              "foo@bar.com",
	"email_verified":     true,
	"preferred_username": "foo",
	"organization": map[string]interface{}{
		"id":   "foo",
		"name": "bar",
	},
	"given_name":   "foo",
	"family_name":  "bar",
	"is_org_admin": true,
}

// helper func to copy claims to avoid mutating the original
func copyClaims(claims jwt.MapClaims) jwt.MapClaims {
	copy := jwt.MapClaims{}
	for k, v := range claims {
		copy[k] = v
	}
	return copy
}

func TestCommercialOCMStandardClaimsValid(t *testing.T) {
	RegisterTestingT(t)

	claims := copyClaims(CommercialValidClaims)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(VerifyOCMClaims(claims)).To(BeTrue())
	Expect(ocmStandardClaims.Audience[0]).To(Equal(claims["aud"]))
	Expect(*ocmStandardClaims.Subject).To(Equal(claims["sub"]))
	Expect(*ocmStandardClaims.IdentityProvider).To(Equal(claims["idp"]))
	Expect(*ocmStandardClaims.Issuer).To(Equal(claims["iss"]))
	Expect(*ocmStandardClaims.Locale).To(Equal(claims["locale"]))
	Expect(*ocmStandardClaims.Scope).To(Equal(claims["scope"]))
	Expect(*ocmStandardClaims.Email).To(Equal(claims["email"]))
	Expect(ocmStandardClaims.EmailVerified).To(Equal(claims["email_verified"]))
	Expect(*ocmStandardClaims.Username).To(Equal(claims["preferred_username"]))
	Expect(ocmStandardClaims.CognitoUsername).To(BeNil())
	Expect(*ocmStandardClaims.Organization.ID).To(Equal(claims["organization"].(map[string]interface{})["id"]))
	Expect(*ocmStandardClaims.Organization.AccountNumber).To(Equal(claims["organization"].(map[string]interface{})["account_number"]))
	Expect(*ocmStandardClaims.FirstName).To(Equal(claims["given_name"]))
	Expect(*ocmStandardClaims.LastName).To(Equal(claims["family_name"]))
	Expect(*ocmStandardClaims.RHITUserID).To(Equal(claims["user_id"]))
	Expect(*ocmStandardClaims.RHCreatorID).To(Equal(claims["rh-user-id"]))
	Expect(ocmStandardClaims.Access.Roles).To(Equal(claims["realm_access"].(map[string]interface{})["roles"]))
}
func TestCommercialOCMStandardClaimsInvalid(t *testing.T) {
	RegisterTestingT(t)

	// Invalid audience
	claims := copyClaims(CommercialValidClaims)
	claims["aud"] = "foo"
	Expect(VerifyOCMClaims(claims)).To(BeFalse())

	// Verify claims still parse without error
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessScopeClaims := OCMStandardClaims{}
	err := accessScopeClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	// Invalid scope
	claims = copyClaims(CommercialValidClaims)
	claims["scope"] = "foo"
	Expect(VerifyOCMClaims(claims)).To(BeFalse())

	// Verify claims still parse without error
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessScopeClaims = OCMStandardClaims{}
	err = accessScopeClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())
}

func TestCognitoFedRampOCMStandardClaimsValid(t *testing.T) {
	RegisterTestingT(t)

	claims := copyClaims(CognitoFedRampValidClaims)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(VerifyOCMClaims(claims)).To(BeTrue())
	Expect(*ocmStandardClaims.Subject).To(Equal(claims["sub"]))
	Expect(*ocmStandardClaims.Issuer).To(Equal(claims["iss"]))
	Expect(*ocmStandardClaims.Scope).To(Equal(claims["scope"]))
	Expect(*ocmStandardClaims.ClientID).To(Equal(claims["client_id"]))
	Expect(*ocmStandardClaims.CognitoUsername).To(Equal(claims["username"]))
	Expect(ocmStandardClaims.Groups).To(Equal(claims["cognito:groups"]))
}
func TestKeycloakFedRampOCMStandardClaimsValid(t *testing.T) {
	RegisterTestingT(t)

	claims := copyClaims(KeycloakFedRampValidClaims)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(*ocmStandardClaims.Subject).To(Equal(claims["sub"]))
	Expect(*ocmStandardClaims.Issuer).To(Equal(claims["iss"]))
	Expect(*ocmStandardClaims.Scope).To(Equal(claims["scope"]))
	Expect(*ocmStandardClaims.Email).To(Equal(claims["email"]))
	Expect(ocmStandardClaims.EmailVerified).To(Equal(claims["email_verified"]))
	Expect(*ocmStandardClaims.Username).To(Equal(claims["preferred_username"]))
	Expect(*ocmStandardClaims.Organization.ID).To(Equal(claims["organization"].(map[string]interface{})["id"]))
	Expect(*ocmStandardClaims.Organization.Name).To(Equal(claims["organization"].(map[string]interface{})["name"]))
	Expect(*ocmStandardClaims.FirstName).To(Equal(claims["given_name"]))
	Expect(*ocmStandardClaims.LastName).To(Equal(claims["family_name"]))
	Expect(ocmStandardClaims.IsOrgAdmin).To(Equal(claims["is_org_admin"]))
}

func TestMultiValueAudience(t *testing.T) {
	RegisterTestingT(t)

	claims := copyClaims(CommercialValidClaims)
	claims["aud"] = []string{"foo", "bar"}
	Expect(VerifyOCMClaims(claims)).To(BeFalse()) //invalid audience (no api.ocm)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(ocmStandardClaims.Audience).To(Equal([]string{"foo", "bar"}))

	claims["aud"] = []string{"cloud-services", "api.ocm"}
	Expect(VerifyOCMClaims(claims)).To(BeTrue()) //valid audience

	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims = OCMStandardClaims{}
	err = ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(ocmStandardClaims.Audience).To(Equal([]string{"cloud-services", "api.ocm"}))
}

func TestCommercialOCMStandardClaimsValid_ServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	claims := copyClaims(CommercialServiceAccountValidClaims)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(VerifyOCMClaims(claims)).To(BeTrue())
	Expect(ocmStandardClaims.Audience[0]).To(Equal(claims["aud"]))
	Expect(*ocmStandardClaims.Subject).To(Equal(claims["sub"]))
	Expect(*ocmStandardClaims.Issuer).To(Equal(claims["iss"]))
	Expect(*ocmStandardClaims.Locale).To(Equal(claims["locale"]))
	Expect(*ocmStandardClaims.Scope).To(Equal(claims["scope"]))
	Expect(*ocmStandardClaims.Username).To(Equal(claims["preferred_username"]))
	Expect(*ocmStandardClaims.ClientID).To(Equal(claims["clientId"]))
	Expect(ocmStandardClaims.CognitoUsername).To(BeNil())

	// Hard-coded audience
	claims = copyClaims(CommercialServiceAccountValidClaims)
	claims["aud"] = "service-account-foo"
	Expect(VerifyOCMClaims(claims)).To(BeTrue())
}

func TestCommercialOCMStandardClaimsValid_OrgServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	claims := copyClaims(CommercialOrgServiceAccountValidClaims)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(VerifyOCMClaims(claims)).To(BeTrue())
	Expect(ocmStandardClaims.Audience[0]).To(Equal(claims["aud"]))
	Expect(*ocmStandardClaims.Subject).To(Equal(claims["sub"]))
	Expect(*ocmStandardClaims.Issuer).To(Equal(claims["iss"]))
	Expect(*ocmStandardClaims.Scope).To(Equal(claims["scope"]))
	Expect(*&ocmStandardClaims.EmailVerified).To(Equal(claims["email_verified"]))
	Expect(*ocmStandardClaims.Username).To(Equal(claims["preferred_username"]))
	Expect(*ocmStandardClaims.ClientID).To(Equal(claims["clientId"]))
	Expect(*ocmStandardClaims.Organization.ID).To(Equal(claims["organization"].(map[string]interface{})["id"]))
	Expect(*ocmStandardClaims.RHCreatorID).To(Equal(claims["rh-user-id"]))
}

func TestCommercialOCMStandardClaimsInvalid_OrgServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	claims := copyClaims(CommercialOrgServiceAccountValidClaims)
	claims["scope"] = ""

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(VerifyOCMClaims(claims)).To(BeFalse())
}

func TestClientIdFallback(t *testing.T) {
	RegisterTestingT(t)

	claims := copyClaims(CommercialValidClaims)
	claims["client_id"] = "foo"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(VerifyOCMClaims(claims)).To(BeTrue())
	Expect(*ocmStandardClaims.ClientID).To(Equal(claims["client_id"]))

	claims["clientId"] = "bar"

	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims = OCMStandardClaims{}
	err = ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(VerifyOCMClaims(claims)).To(BeTrue())
	Expect(*ocmStandardClaims.ClientID).To(Equal(claims["clientId"]))
}
