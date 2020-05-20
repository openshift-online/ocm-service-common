package test

import (
	"testing"

	. "github.com/onsi/gomega"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

func TestRun(t *testing.T) {
	RegisterTestingT(t)

	testSuiteSpec := NewTestSuiteSpec()
	suite, err := BuildTestSuite(testSuiteSpec)
	if err != nil {
		t.Errorf("Could not build test suite.")
	}

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
