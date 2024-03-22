package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	"github.com/golang-jwt/jwt/v4"
)

func TestValidAudienceString(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("cloud-services", "")

	middleware := NewTokenValidationMiddleware(DefaultApprovedAudiences, nil, nil)
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateAudience(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestValidAudienceSlice(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx([]string{"cloud-services", "other-audience"}, "")

	middleware := NewTokenValidationMiddleware(DefaultApprovedAudiences, nil, nil)
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateAudience(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestValidNoApprovedAudience(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx([]string{"cloud-services", "other-audience"}, "")

	// Validates that no validation is performed when approvedAudiences is empty
	middleware := NewTokenValidationMiddleware(nil, nil, nil)
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateAudience(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestInvalidAudience(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx([]string{"invalid-audience"}, "")

	middleware := NewTokenValidationMiddleware(DefaultApprovedAudiences, nil, nil)
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateAudience(ctx)
	Expect(err).To(HaveOccurred())
	Expect(err).To(Equal(ErrInvalidAudience))
}

func TestServiceAcctAudience(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateServiceAcctTokenCtx("client_id", "openid")

	middleware := NewTokenValidationMiddleware(DefaultApprovedAudiences, nil, nil)
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateAudience(ctx)
	Expect(err).NotTo(HaveOccurred())

	ctx = generateServiceAcctTokenCtx("clientId", "openid")

	err = middleware.ValidateAudience(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestValidRequiredScopes(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("", "openid api.ocm")

	middleware := NewTokenValidationMiddleware(nil, nil, []string{"api.ocm"})
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateScopes(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestInvalidRequiredScopes(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("", "openid")

	middleware := NewTokenValidationMiddleware(nil, nil, []string{"api.ocm"})
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateScopes(ctx)
	Expect(err).To(HaveOccurred())
	Expect(errors.Unwrap(err)).To(Equal(ErrMissingRequiredScopes))
}

// Validates that the token does not contain any of the denyScopes
func TestValidDenyScopes(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("", "openid api.ocm")

	middleware := NewTokenValidationMiddleware(nil, []string{"offline_access"}, nil)
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateScopes(ctx)
	Expect(err).NotTo(HaveOccurred())
}

// Validates that the token is rejected if it contains any of the denyScopes
func TestInvalidDenyScopes(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("", "openid api.ocm offline_access")

	middleware := NewTokenValidationMiddleware(nil, []string{"offline_access"}, nil)
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateScopes(ctx)
	Expect(err).To(HaveOccurred())
	Expect(errors.Unwrap(err)).To(Equal(ErrUnauthorizedScopes))
}

func TestServiceAccountNoScopeValidation(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateServiceAcctTokenCtx("client_id", "openid")

	middleware := NewTokenValidationMiddleware(nil, nil, []string{"api.ocm"})
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateScopes(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestServiceAccountWithScopeValidationPass(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateServiceAcctTokenCtx("clientId", "openid api.ocm")

	middleware := NewTokenValidationMiddleware(nil, nil, []string{"api.ocm"})
	Expect(middleware).NotTo(BeNil())
	middleware.enforceServiceAccountScopes = true

	err := middleware.ValidateScopes(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestServiceAccountWithScopeValidationFail(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateServiceAcctTokenCtx("client_id", "openid")

	middleware := NewTokenValidationMiddleware(nil, nil, []string{"api.ocm"})
	Expect(middleware).NotTo(BeNil())
	middleware.enforceServiceAccountScopes = true

	err := middleware.ValidateScopes(ctx)
	Expect(err).To(HaveOccurred())
	Expect(errors.Unwrap(err)).To(Equal(ErrMissingRequiredScopes))
}

func TestValidateAll(t *testing.T) {
	RegisterTestingT(t)

	ctx := generateBasicTokenCtx("", "openid api.ocm")

	middleware := NewTokenValidationMiddleware([]string{""}, []string{"offline_access"}, []string{"api.ocm"})
	Expect(middleware).NotTo(BeNil())

	err := middleware.ValidateAll(ctx)
	Expect(err).NotTo(HaveOccurred())
}

func TestMiddlewareValidate(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})
	middleware := NewTokenValidationMiddleware([]string{""}, []string{"offline_access"}, []string{"api.ocm"}).Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	ctx := generateBasicTokenCtx("", "openid api.ocm")
	request = request.WithContext(ctx)
	middleware.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
}

func TestMiddlewareValidateNoContext(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fail() // Should not be called
	})
	middleware := NewTokenValidationMiddleware([]string{""}, []string{"offline_access"}, []string{"api.ocm"}).Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	middleware.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	Expect(recorder.Body.String()).ToNot(BeEmpty())
}

func TestMiddlewareValidateWithCallbackPassthrough(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})
	middleware := NewTokenValidationMiddleware([]string{""}, []string{"offline_access"}, []string{"api.ocm"})
	middleware.callbackFn = callback

	handler := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	ctx := generateBasicTokenCtx("", "openid api.ocm")
	request = request.WithContext(ctx)
	handler.ServeHTTP(recorder, request)
	Expect(recorder.Code).To(Equal(http.StatusOK))
}

func TestMiddlewareValidateWithCallbackError(t *testing.T) {
	RegisterTestingT(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Expect(r.Method).To(Equal(http.MethodGet))
	})
	middleware := NewTokenValidationMiddleware([]string{""}, []string{"offline_access"}, []string{"api.ocm"})
	middleware.callbackFn = callbackExpectError

	handler := middleware.Handler(nextHandler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	ctx := generateBasicTokenCtx("", "openid")
	request = request.WithContext(ctx)
	handler.ServeHTTP(recorder, request)
	// Test the status came from the callback and not the middleware
	Expect(recorder.Code).To(Equal(http.StatusForbidden))
}

func generateBasicTokenCtx(aud interface{}, scope string) context.Context {
	claims := jwt.MapClaims{
		"aud":   aud,
		"scope": scope,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return context.WithValue(context.Background(), "token", token)
}

// Generates a service account token context
// acceptable client id key is "client_id" (standard) or "clientId" (legacy fallback)
func generateServiceAcctTokenCtx(clientIdKey string, scope string) context.Context {
	claims := jwt.MapClaims{
		clientIdKey: "1234",
		"scope":     scope,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return context.WithValue(context.Background(), "token", token)
}

func callback(w http.ResponseWriter, r *http.Request, err error) {
}

func callbackExpectError(w http.ResponseWriter, r *http.Request, err error) {
	Expect(err).To(HaveOccurred())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(err.Error()))
}
