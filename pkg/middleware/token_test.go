package middleware

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	test "github.com/openshift-online/ocm-service-common/pkg/test"
)

const testUsername = "mturansk.openshift"

func generateAccountListJSON(accounts []v1.Account) string {
	items := make([]string, len(accounts))

	for i, acc := range accounts {
		items[i] =
			fmt.Sprintf(`{
				"created_at":"2024-05-20T14:48:28.2931Z",
				"email": "%[1]s",
				"href":"/api/accounts_mgmt/v1/accounts/%[2]s",
				"id":"%[2]s",
				"kind":"Account",
				"organization": {
				  "id":"%[3]s"
				},
				"service_account":false,
				"updated_at":"2024-05-20T14:48:28.2931Z",
				"username":"%[4]s"
			}`, acc.Email(), acc.ID(), acc.Organization().ID(), acc.Username())
	}

	return `{
        "items": [` + strings.Join(items, ",") + `]
    }`
}

func generateTokenAuthorizationJSON(account v1.Account) string {
	return fmt.Sprintf(`{
		"account": {
			"id": "%s",
			"username": "%s"
		}
	}`, account.ID(), account.Username())
}

func TestTokenMiddlewareSuccess(t *testing.T) {
	RegisterTestingT(t)
	apiServer := MakeTCPServer()
	saToken, err := authentication.TokenFromContext(generateBasicTokenCtx("openid", "111111")) // service account mock
	Expect(err).NotTo(HaveOccurred())
	signedSaToken, _ := saToken.SignedString([]byte("secret"))
	ssoServer := MakeTCPServer()
	ssoServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"access_token": "%s"}`, signedSaToken)),
	)
	testSuiteSpec := test.NewMockTestSuiteSpec(apiServer.URL(), ssoServer.URL())
	testSuiteSpec.DefaultAccountID = "123"

	accountID := test.GetAccountID(testSuiteSpec)
	acc, err := v1.NewAccount().ID(accountID).
		Organization(v1.NewOrganization().ID("123")).
		Email("example@redhat.com").Username(testUsername).Build()
	Expect(err).ToNot(HaveOccurred())
	apiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, generateAccountListJSON([]v1.Account{*acc})),
		RespondWithJSON(http.StatusOK, generateTokenAuthorizationJSON(*acc)),
	)
	suite, err := test.BuildTestSuite(testSuiteSpec)
	Expect(suite).NotTo(BeNil())
	Expect(err).ToNot(HaveOccurred())

	tokenMiddleware, err := NewTokenAuthMiddleware(suite.Connection())
	Expect(err).NotTo(HaveOccurred())
	response, err := suite.Connection().AccountsMgmt().V1().
		RegistryCredentials().List().Size(1).Search(fmt.Sprintf("account_id = '%s'", accountID)).Send()
	Expect(err).NotTo(HaveOccurred())
	regCreds, _ := response.GetItems()
	Expect(regCreds.Empty()).To(BeFalse())
	regCred := regCreds.Get(0)

	username := regCred.Username()
	token := regCred.Token()
	tokenAuth := base64.StdEncoding.EncodeToString([]byte(username + ":" + token))

	headers := make(map[string][]string)
	authHeader := fmt.Sprintf("AccessToken b82847e7-dde7-4fb5-a55a-ab00b7b7dc62:%s", tokenAuth)
	headers["Authorization"] = []string{authHeader}
	tokenAuthFoundAccountID, foundUsername := tokenMiddleware.Authenticate(context.Background(), headers)

	// Found account matching registry credential in Authorization header
	Expect(tokenAuthFoundAccountID).To(Equal(accountID))
	Expect(foundUsername).To(Equal(testUsername))
}

func TestTokenMiddlewareFailure(t *testing.T) {
	RegisterTestingT(t)

	apiServer := MakeTCPServer()
	saToken, err := authentication.TokenFromContext(generateBasicTokenCtx("openid", "111111")) // service account mock
	Expect(err).NotTo(HaveOccurred())
	signedSaToken, _ := saToken.SignedString([]byte("secret"))
	ssoServer := MakeTCPServer()
	ssoServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"access_token": "%s"}`, signedSaToken)),
	)
	testSuiteSpec := test.NewMockTestSuiteSpec(apiServer.URL(), ssoServer.URL())
	testSuiteSpec.DefaultAccountID = "123"
	suite, err := test.BuildTestSuite(testSuiteSpec)
	if err != nil {
		t.Errorf("Could not build test suite: %v", err)
	}

	tokenMiddleware, err := NewTokenAuthMiddleware(suite.Connection())
	Expect(err).NotTo(HaveOccurred())
	headers := make(map[string][]string)

	// Invalid AccessToken
	headers["Authorization"] = []string{"AccessToken invalid-nonsense"}
	missingAccountId, missingUsername := tokenMiddleware.Authenticate(context.Background(), headers)

	// No account found
	Expect(missingAccountId).To(BeEmpty())
	Expect(missingUsername).To(BeEmpty())

	// No AccessToken Header
	headers["Authorization"] = []string{""}
	missingAccountId, missingUsername = tokenMiddleware.Authenticate(context.Background(), headers)

	// No account found
	Expect(missingAccountId).To(BeEmpty())
	Expect(missingUsername).To(BeEmpty())
}
