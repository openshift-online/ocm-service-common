package test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

func TestRun(t *testing.T) {
	RegisterTestingT(t)

	apiServer := MakeTCPServer()
	saToken, err := authentication.TokenFromContext(generateBasicTokenCtx("openid", "111111")) // service account mock
	Expect(err).NotTo(HaveOccurred())
	signedSaToken, _ := saToken.SignedString([]byte("secret"))
	ssoServer := MakeTCPServer()
	ssoServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"access_token": "%s"}`, signedSaToken)),
	)
	testSuiteSpec := NewMockTestSuiteSpec(apiServer.URL(), ssoServer.URL())
	testSuiteSpec.DefaultAccountID = "123"
	Expect(err).ToNot(HaveOccurred())
	emptyResponse := `{}`
	apiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK, emptyResponse),
		RespondWithJSON(http.StatusOK, emptyResponse),
		RespondWithJSON(http.StatusOK, emptyResponse),
		RespondWithJSON(http.StatusOK, emptyResponse),
	)
	suite, err := BuildTestSuite(testSuiteSpec)
	Expect(suite).NotTo(BeNil())
	Expect(err).ToNot(HaveOccurred())

	testCasesWithoutError := []*TestCase{
		{
			Name:   "GET /api/clusters_mgmt/v1/",
			Labels: []string{"test"},
			TestFunc: func(s TestState) (*sdk.Response, error) {
				return suite.Connection().Get().Path("/api/clusters_mgmt/v1/").Send()
			},
			ResponseAssertions: []ResponseAssertion{
				AssertResponseStatusOK(),
			},
		},
		{
			Name:   "GET /api/accounts_mgmt/v1/",
			Labels: []string{"test", "read-only"},
			TestFunc: func(s TestState) (*sdk.Response, error) {
				return suite.Connection().Get().Path("/api/accounts_mgmt/v1/").Send()
			},
			ResponseAssertions: []ResponseAssertion{
				AssertResponseStatusOK(),
			},
		},
	}

	suite.AddTestCases(testCasesWithoutError)

	testCfg := &TestConfig{
		SampleCount: 2,
		Labels:      []string{"test", "read-only"},
	}

	resultSet := suite.Run(testCfg)
	Expect(resultSet).ToNot(BeNil())
	Expect(len(resultSet)).To(Equal(len(testCasesWithoutError)))

	for _, results := range resultSet {
		Expect(len(results)).To(Equal(testCfg.SampleCount))
		for _, res := range results {
			Expect(res.Error).To(BeNil())
			Expect(res.Size).ToNot(BeZero())
		}
	}
}

func generateBasicTokenCtx(scope string, orgId string) context.Context {
	claims := jwt.MapClaims{
		"scope":  scope,
		"org_id": orgId,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return authentication.ContextWithToken(context.Background(), token)
}
