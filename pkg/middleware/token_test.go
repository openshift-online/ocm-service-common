package middleware

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	test "gitlab.cee.redhat.com/service/ocm-common/pkg/test"
)

func TestTokenMiddlewareSuccess(t *testing.T) {
	RegisterTestingT(t)

	testSuiteSpec := test.NewTestSuiteSpec()
	suite, err := test.BuildTestSuite(testSuiteSpec)
	if err != nil {
		t.Errorf("Could not build test suite.")
	}

	tokenMiddleware, err := NewTokenAuthMiddleware(suite.Connection())
	Expect(err).NotTo(HaveOccurred())

	accountID := test.GetAccountID(testSuiteSpec)
	response, err := suite.Connection().AccountsMgmt().V1().RegistryCredentials().List().Size(1).Search(fmt.Sprintf("account_id = '%s'", accountID)).Send()
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
	ctx := tokenMiddleware.Authenticate(context.Background(), headers)
	tokenAuthFoundAccountID := ctx.Value(ContextAccountIDKey)

	// Found account matching registry credential in Authorization header
	Expect(tokenAuthFoundAccountID).To(Equal(accountID))
}

func TestTokenMiddlewareFailure(t *testing.T) {
	RegisterTestingT(t)

	testSuiteSpec := test.NewTestSuiteSpec()
	suite, err := test.BuildTestSuite(testSuiteSpec)
	if err != nil {
		t.Errorf("Could not build test suite.")
	}

	tokenMiddleware, err := NewTokenAuthMiddleware(suite.Connection())
	Expect(err).NotTo(HaveOccurred())
	headers := make(map[string][]string)

	// Invalid AccessToken
	headers["Authorization"] = []string{"AccessToken invalid-nonsense"}
	ctx := tokenMiddleware.Authenticate(context.Background(), headers)
	missingAccountId := ctx.Value(ContextAccountIDKey)

	// No account found
	Expect(missingAccountId).To(BeNil())

	// No AccessToken Header
	headers["Authorization"] = []string{""}
	ctx = tokenMiddleware.Authenticate(context.Background(), headers)
	missingAccountId = ctx.Value(ContextAccountIDKey)

	// No account found
	Expect(missingAccountId).To(BeNil())
}
