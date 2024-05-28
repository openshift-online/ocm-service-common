package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	test "gitlab.cee.redhat.com/service/ocm-common/pkg/test"
	"golang.org/x/net/context"

	"github.com/golang-jwt/jwt/v4"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

func TestValidRequiredScopes(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("openid api.ocm", "123456")

	middleware := TokenScopeValidationMiddleware{
		RequiredScopes: []string{"api.ocm"},
	}

	err := middleware.ValidateScopes(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestInvalidRequiredScopes(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("openid", "123456")

	middleware := TokenScopeValidationMiddleware{
		RequiredScopes: []string{"api.ocm"},
	}

	err := middleware.ValidateScopes(ctx)
	Expect(err).To(HaveOccurred())
	Expect(errors.Unwrap(err)).To(Equal(ErrMissingRequiredScopes))
}

// Validates that the token does not contain any of the denyScopes
func TestValidDenyScopes(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("openid api.ocm", "123456")

	middleware := TokenScopeValidationMiddleware{
		DenyScopes: []string{"offline_access"},
	}

	err := middleware.ValidateScopes(ctx)
	Expect(err).NotTo(HaveOccurred())
}

// Validates that the token is rejected if it contains any of the denyScopes
func TestInvalidDenyScopes(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("openid api.ocm offline_access", "123456")

	middleware := TokenScopeValidationMiddleware{
		DenyScopes: []string{"offline_access"},
	}

	err := middleware.ValidateScopes(ctx)
	Expect(err).To(HaveOccurred())
	Expect(errors.Unwrap(err)).To(Equal(ErrUnauthorizedScopes))
}

func TestServiceAccountNoScopeValidation(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateServiceAcctTokenCtx("client_id", "openid")

	middleware := TokenScopeValidationMiddleware{
		DenyScopes: []string{"offline_access"},
	}

	err := middleware.ValidateScopes(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestServiceAccountWithScopeValidationPass(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateServiceAcctTokenCtx("clientId", "openid api.ocm")

	middleware := TokenScopeValidationMiddleware{
		DenyScopes:                  []string{"offline_access"},
		EnforceServiceAccountScopes: true,
	}

	err := middleware.ValidateScopes(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestServiceAccountWithScopeValidationFail(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateServiceAcctTokenCtx("client_id", "openid")

	middleware := TokenScopeValidationMiddleware{
		RequiredScopes:              []string{"api.ocm"},
		EnforceServiceAccountScopes: true,
	}

	err := middleware.ValidateScopes(ctx)
	Expect(err).To(HaveOccurred())
	Expect(errors.Unwrap(err)).To(Equal(ErrMissingRequiredScopes))
}

func TestValidateAll(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("openid api.ocm", "123456")

	middleware := TokenScopeValidationMiddleware{
		DenyScopes:     []string{"offline_access"},
		RequiredScopes: []string{"api.ocm"},
	}

	err := middleware.ValidateAll(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestMiddlewareValidate(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})

	middleware := TokenScopeValidationMiddleware{
		DenyScopes:     []string{"offline_access"},
		RequiredScopes: []string{"api.ocm"},
	}

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	ctx := generateBasicTokenCtx("openid api.ocm", "123456")
	request = request.WithContext(ctx)
	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
}

func TestMiddlewareValidateAllowMissingTokens(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})
	middleware := TokenScopeValidationMiddleware{
		DenyScopes:     []string{"offline_access"},
		RequiredScopes: []string{"api.ocm"},
	}

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
}

func TestMiddlewareValidateErrorOnMissingTokens(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fail() // Should not be called
	})
	middleware := TokenScopeValidationMiddleware{
		ErrorOnMissingToken: true,
		DenyScopes:          []string{"offline_access"},
		RequiredScopes:      []string{"api.ocm"},
	}

	router := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
}

func TestMiddlewareValidateWithCallbackPassthrough(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})
	middleware := TokenScopeValidationMiddleware{
		DenyScopes:     []string{"offline_access"},
		RequiredScopes: []string{"api.ocm"},
		CallbackFn:     callback,
	}

	handler := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	ctx := generateBasicTokenCtx("openid api.ocm", "123456")
	request = request.WithContext(ctx)
	handler.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
}

func TestMiddlewareValidateWithCallbackError(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})
	middleware := TokenScopeValidationMiddleware{
		DenyScopes:     []string{"offline_access"},
		RequiredScopes: []string{"api.ocm"},
		CallbackFn:     callbackExpectError,
	}

	handler := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	ctx := generateBasicTokenCtx("openid", "123456")
	request = request.WithContext(ctx)
	handler.ServeHTTP(recorder, request)
	// Test the status came from the callback and not the middleware
	Expect(recorder.Code).To(Equal(http.StatusForbidden))
}

func TestMiddlewareValidateOfflineAccessByOrganization(t *testing.T) {
	RegisterTestingT(t)

	// Setup orgs
	org1, err := v1.NewOrganization().ID("1a2b3c4d5e6f").ExternalID("123456").Build()
	Expect(err).NotTo(HaveOccurred())
	org2, err := v1.NewOrganization().ID("7g8h9i0j1k2l").ExternalID("123457").Build()
	Expect(err).NotTo(HaveOccurred())

	organizations := []v1.Organization{*org1}

	// Mocking AMS responses
	apiServer := MakeTCPServer()
	apiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, generateBasicLabelResponseJSON(organizations)),
		RespondWithJSON(http.StatusOK, generateBasicOrganizationResponseJSON(organizations)),
		RespondWithJSON(http.StatusOK, generateFeatureResponseJSON(FlagEnforceOfflineTokenRestrictions, true)),
	)

	saToken, err := authentication.TokenFromContext(generateBasicTokenCtx("openid", "111111")) // service account mock
	Expect(err).NotTo(HaveOccurred())

	signedSaToken, _ := saToken.SignedString([]byte("secret"))

	// Mocking SSO responses
	ssoServer := MakeTCPServer()
	ssoServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"access_token": "%s"}`, signedSaToken)),
	)

	testSuiteSpec := test.NewMockTestSuiteSpec(apiServer.URL(), ssoServer.URL())
	suite, err := test.BuildTestSuite(testSuiteSpec)
	Expect(suite).NotTo(BeNil())
	Expect(err).To(BeNil())

	middleware := TokenScopeValidationMiddleware{
		Connection:              suite.Connection(),
		PollingIntervalOverride: 3 * time.Second,
		CallbackFn:              callbackExpectError,
	}

	// Valid Base URL & Organizations
	stopPolling := middleware.StartPollingAMSForRestrictedOrgs()
	defer stopPolling()
	Expect(middleware.isOfflineOrgRestrictionsEnabledSafe()).To(BeTrue())
	Expect(middleware.offlineRestrictedOrgs).To(HaveLen(1))
	Expect(middleware.isOrgRestrictedSafe(org1.ExternalID())).To(BeTrue())
	Expect(middleware.isOrgRestrictedSafe(org2.ExternalID())).To(BeFalse()) // Not included yet

	// Add org2 to the list of organizations
	organizations = append(organizations, *org2)
	labelResponse := generateBasicLabelResponseJSON(organizations)
	orgResponse := generateBasicOrganizationResponseJSON(organizations)
	apiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, labelResponse),
		RespondWithJSON(http.StatusOK, orgResponse),
		RespondWithJSON(http.StatusOK, generateFeatureResponseJSON(FlagEnforceOfflineTokenRestrictions, true)),
	)
	time.Sleep(4 * time.Second)
	Expect(middleware.isOfflineOrgRestrictionsEnabledSafe()).To(BeTrue())
	Expect(middleware.getOfflineRestrictedOrgCountSafe()).To(Equal(2))
	Expect(middleware.isOrgRestrictedSafe(org1.ExternalID())).To(BeTrue())
	Expect(middleware.isOrgRestrictedSafe(org2.ExternalID())).To(BeTrue()) // Now included

	// Validate that the middleware is enforcing the org restrictions
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fail() // Should not be called
	})

	// Validate with the offline_access scope
	handler := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	ctx := generateBasicTokenCtx("openid offline_access", org1.ExternalID())
	request = request.WithContext(ctx)
	handler.ServeHTTP(recorder, request)
	// Test the status came from the callback and not the middleware
	Expect(recorder.Code).To(Equal(http.StatusForbidden))
	Expect(recorder.Body.String()).To(ContainSubstring("offline access is restricted for organization"))

	// Validate without the offline_access scope
	middleware.CallbackFn = callback
	nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})
	handler = middleware.Handler(nextHandler)
	ctx = generateBasicTokenCtx("openid", org1.ExternalID())
	request = request.WithContext(ctx)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))

	// Validate offline access works for a non-restricted org
	ctx = generateBasicTokenCtx("openid offline_access", "123458")
	err = middleware.ValidateOfflineAccessByOrg(ctx)
	Expect(err).NotTo(HaveOccurred())

	// Turn off the feature toggle
	apiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, labelResponse),
		RespondWithJSON(http.StatusOK, orgResponse),
		RespondWithJSON(http.StatusOK, generateFeatureResponseJSON(FlagEnforceOfflineTokenRestrictions, false)),
	)
	time.Sleep(4 * time.Second)
	Expect(middleware.getOfflineRestrictedOrgCountSafe()).To(Equal(2))     // Still have two orgs restricted
	Expect(middleware.isOfflineOrgRestrictionsEnabledSafe()).To(BeFalse()) // But the feature toggle is disabled

	// Validate a restricted org can now use an offline token
	ctx = generateBasicTokenCtx("openid", org1.ExternalID())
	request = request.WithContext(ctx)
	handler.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))

}

func TestMiddlewareValidateOfflineAccessByOrganizationCapabilityFailOpen(t *testing.T) {
	RegisterTestingT(t)

	// Mocking AMS responses
	// Capability check fail, feature toggle success
	apiServer := MakeTCPServer()
	apiServer.AppendHandlers(
		RespondWithJSON(http.StatusUnauthorized, `error`), // Capability check
		RespondWithJSON(http.StatusOK, generateFeatureResponseJSON(FlagEnforceOfflineTokenRestrictions, false)),
	)

	// Mocking SSO responses
	saToken, err := authentication.TokenFromContext(generateBasicTokenCtx("openid", "111111")) // service account mock
	Expect(err).NotTo(HaveOccurred())
	signedSaToken, _ := saToken.SignedString([]byte("secret"))
	ssoServer := MakeTCPServer()
	ssoServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"access_token": "%s"}`, signedSaToken)),
	)

	testSuiteSpec := test.NewMockTestSuiteSpec(apiServer.URL(), ssoServer.URL())
	suite, err := test.BuildTestSuite(testSuiteSpec)
	Expect(suite).NotTo(BeNil())
	Expect(err).To(BeNil())

	middleware := TokenScopeValidationMiddleware{
		Connection:              suite.Connection(),
		PollingIntervalOverride: 3 * time.Second,
	}

	// Invalid responses from AMS
	stopPolling := middleware.StartPollingAMSForRestrictedOrgs()
	defer stopPolling()
	Expect(middleware.getOfflineRestrictedOrgCountSafe()).To(Equal(0))

	nextHandlerCalled := false
	// Validate that the middleware is NOT enforcing the org restrictions
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		Expect(r.Method).To(Equal(http.MethodGet))
	})

	// Validate with the offline_access scope
	handler := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	ctx := generateBasicTokenCtx("openid offline_access", "123456")
	request = request.WithContext(ctx)
	handler.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(nextHandlerCalled).To(BeTrue())
	nextHandlerCalled = false

	// Validate without the offline_access scope
	middleware.CallbackFn = callback
	handler = middleware.Handler(nextHandler)
	ctx = generateBasicTokenCtx("openid", "123456")
	request = request.WithContext(ctx)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(nextHandlerCalled).To(BeTrue())
}

func TestMiddlewareValidateOfflineAccessByOrganizationToggleFailOpen(t *testing.T) {
	RegisterTestingT(t)

	// Setup orgs
	restrictedOrg, err := v1.NewOrganization().ID("1a2b3c4d5e6f").ExternalID("123456").Build()
	Expect(err).NotTo(HaveOccurred())

	organizations := []v1.Organization{*restrictedOrg}

	// Mocking AMS responses
	// Capability check success, feature toggle fail
	apiServer := MakeTCPServer()
	apiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, generateBasicLabelResponseJSON(organizations)),
		RespondWithJSON(http.StatusOK, generateBasicOrganizationResponseJSON(organizations)),
		RespondWithJSON(http.StatusUnauthorized, `error`), // Feature flag check
	)

	// Mocking SSO responses
	saToken, err := authentication.TokenFromContext(generateBasicTokenCtx("openid", "111111")) // service account mock
	Expect(err).NotTo(HaveOccurred())
	signedSaToken, _ := saToken.SignedString([]byte("secret"))
	ssoServer := MakeTCPServer()
	ssoServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"access_token": "%s"}`, signedSaToken)),
	)

	testSuiteSpec := test.NewMockTestSuiteSpec(apiServer.URL(), ssoServer.URL())
	suite, err := test.BuildTestSuite(testSuiteSpec)
	Expect(suite).NotTo(BeNil())
	Expect(err).To(BeNil())

	middleware := TokenScopeValidationMiddleware{
		Connection:              suite.Connection(),
		PollingIntervalOverride: 3 * time.Second,
	}

	// Invalid responses from AMS
	stopPolling := middleware.StartPollingAMSForRestrictedOrgs()
	defer stopPolling()
	Expect(middleware.getOfflineRestrictedOrgCountSafe()).To(Equal(1))

	nextHandlerCalled := false
	// Validate that the middleware is NOT enforcing the org restrictions
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		Expect(r.Method).To(Equal(http.MethodGet))
	})

	// Validate with the offline_access scope
	handler := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	ctx := generateBasicTokenCtx("openid offline_access", restrictedOrg.ExternalID())
	request = request.WithContext(ctx)
	handler.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(nextHandlerCalled).To(BeTrue())
	nextHandlerCalled = false

	// Validate without the offline_access scope
	middleware.CallbackFn = callback
	handler = middleware.Handler(nextHandler)
	ctx = generateBasicTokenCtx("openid", restrictedOrg.ExternalID())
	request = request.WithContext(ctx)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(nextHandlerCalled).To(BeTrue())
}

// Tests that if the middleware is applied on an endpoint using cloud.openshift.com pull secret authentication
// that we do not fail when `ErrorOnMissingToken` is false
func TestMiddlewareGracefulHandlingAccessTokenPullSecret(t *testing.T) {
	RegisterTestingT(t)

	// Setup orgs
	org1, err := v1.NewOrganization().ID("1a2b3c4d5e6f").ExternalID("123456").Build()
	Expect(err).NotTo(HaveOccurred())

	organizations := []v1.Organization{*org1}

	// Mocking AMS responses
	apiServer := MakeTCPServer()
	apiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, generateBasicLabelResponseJSON(organizations)),
		RespondWithJSON(http.StatusOK, generateBasicOrganizationResponseJSON(organizations)),
		RespondWithJSON(http.StatusOK, generateFeatureResponseJSON(FlagEnforceOfflineTokenRestrictions, true)),
	)

	// Mocking SSO responses
	saToken, err := authentication.TokenFromContext(generateBasicTokenCtx("openid", "111111")) // service account mock
	Expect(err).NotTo(HaveOccurred())
	signedSaToken, _ := saToken.SignedString([]byte("secret"))
	ssoServer := MakeTCPServer()
	ssoServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"access_token": "%s"}`, signedSaToken)),
	)

	testSuiteSpec := test.NewMockTestSuiteSpec(apiServer.URL(), ssoServer.URL())
	suite, err := test.BuildTestSuite(testSuiteSpec)
	Expect(suite).NotTo(BeNil())
	Expect(err).To(BeNil())

	middleware := TokenScopeValidationMiddleware{
		Connection:              suite.Connection(),
		PollingIntervalOverride: 3 * time.Second,
		CallbackFn:              callback,
	}

	stopPolling := middleware.StartPollingAMSForRestrictedOrgs()
	defer stopPolling()

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
	})

	handler := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	clusterUUID := uuid.New()
	base64Secret := "Zm9vYmFyLWZvb2Jhcg==" // foobar-foobar
	request.Header.Set("Authorization", fmt.Sprintf("AccessToken %s:%s", clusterUUID, base64Secret))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	Expect(nextHandlerCalled).To(BeTrue())
}

func generateBasicTokenCtx(scope string, orgId string) context.Context {
	claims := jwt.MapClaims{
		"scope":  scope,
		"org_id": orgId,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return authentication.ContextWithToken(context.Background(), token)
}

// Generates a service account token context
// acceptable client id key is "client_id" (standard) or "clientId" (legacy fallback)
func generateServiceAcctTokenCtx(clientIdKey string, scope string) context.Context {
	claims := jwt.MapClaims{
		clientIdKey: "1234",
		"scope":     scope,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return authentication.ContextWithToken(context.Background(), token)
}

func generateBasicLabelResponseJSON(orgs []v1.Organization) string {
	items := make([]string, len(orgs))

	for i, org := range orgs {
		items[i] = `{
            "internal":true,
            "key":"` + OfflineAccessCapabilityKey + `",
            "organization_id":"` + org.ID() + `",
            "value":"true"
        }`
	}

	return `{
        "items": [` + strings.Join(items, ",") + `]
    }`
}
func generateBasicOrganizationResponseJSON(orgs []v1.Organization) string {
	items := make([]string, len(orgs))

	for i, org := range orgs {
		items[i] = `{
            "external_id":"` + org.ExternalID() + `",
            "id":"` + org.ID() + `"
        }`
	}

	return `{
        "items": [` + strings.Join(items, ",") + `]
    }`
}

func generateFeatureResponseJSON(flag string, value bool) string {
	return `{
		"enabled": ` + fmt.Sprintf("%t", value) + `,
		"feature_id": "` + flag + `"
	}`
}

func callback(w http.ResponseWriter, r *http.Request, err error) {
}

func callbackExpectError(w http.ResponseWriter, r *http.Request, err error) {
	Expect(err).To(HaveOccurred())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(err.Error()))
}
