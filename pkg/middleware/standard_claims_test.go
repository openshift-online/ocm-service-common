package middleware

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/golang-jwt/jwt/v4"
)

var CommercialValidClaims = jwt.MapClaims{
	"aud":                OCMAudience,
	"sub":                "foo",
	"idp":                "https://foo.bar",
	"iss":                "https://sso.redhat.com/auth/realms/redhat-external",
	"locale":             "en_US",
	"scope":              fmt.Sprintf("openid %s", OCMAccessScope),
	"client_id":          "foo",
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
	"scope":              "openid",
	"email":              "foo@bar.com",
	"email_verified":     true,
	"preferred_username": "foo",
	"organization": map[string]interface{}{
		"id":   "foo",
		"name": "bar",
	},
	"given_name":  "foo",
	"family_name": "bar",
}

func TestCommercialOCMStandardClaimsValid(t *testing.T) {
	RegisterTestingT(t)

	claims := CommercialValidClaims
	Expect(JWTContainsOCMAccessScope(claims)).To(BeTrue())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(ocmStandardClaims.CommercialIsValid()).To(BeTrue())
	Expect(*ocmStandardClaims.Audience).To(Equal(claims["aud"]))
	Expect(*ocmStandardClaims.Subject).To(Equal(claims["sub"]))
	Expect(*ocmStandardClaims.IdentityProvider).To(Equal(claims["idp"]))
	Expect(*ocmStandardClaims.Issuer).To(Equal(claims["iss"]))
	Expect(*ocmStandardClaims.Locale).To(Equal(claims["locale"]))
	Expect(*ocmStandardClaims.Scope).To(Equal(claims["scope"]))
	Expect(*ocmStandardClaims.ClientID).To(Equal(claims["client_id"]))
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
	claims := CommercialValidClaims
	claims["aud"] = "foo"
	Expect(JWTContainsOCMAccessScope(claims)).To(BeTrue())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessScopeClaims := OCMStandardClaims{}
	err := accessScopeClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	ok, err := accessScopeClaims.CommercialIsValid()
	Expect(ok).To(BeFalse())
	Expect(err).To(HaveOccurred())

	// Invalid scope
	claims["scope"] = "foo"
	Expect(JWTContainsOCMAccessScope(claims)).To(BeFalse())

	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessScopeClaims = OCMStandardClaims{}
	err = accessScopeClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	ok, err = accessScopeClaims.CommercialIsValid()
	Expect(ok).To(BeFalse())
	Expect(err).To(HaveOccurred())
}

func TestCognitoFedRampOCMStandardClaimsValid(t *testing.T) {
	RegisterTestingT(t)

	claims := CognitoFedRampValidClaims

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ocmStandardClaims := OCMStandardClaims{}
	err := ocmStandardClaims.UnmarshalFromToken(token)
	Expect(err).NotTo(HaveOccurred())

	Expect(*ocmStandardClaims.Subject).To(Equal(claims["sub"]))
	Expect(*ocmStandardClaims.Issuer).To(Equal(claims["iss"]))
	Expect(*ocmStandardClaims.Scope).To(Equal(claims["scope"]))
	Expect(*ocmStandardClaims.ClientID).To(Equal(claims["client_id"]))
	Expect(*ocmStandardClaims.CognitoUsername).To(Equal(claims["username"]))
	Expect(ocmStandardClaims.Groups).To(Equal(claims["cognito:groups"]))
}
func TestKeycloakFedRampOCMStandardClaimsValid(t *testing.T) {
	RegisterTestingT(t)

	claims := KeycloakFedRampValidClaims

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
}
