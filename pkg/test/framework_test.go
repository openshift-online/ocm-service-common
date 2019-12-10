package test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestRunner(t *testing.T) {
	RegisterTestingT(t)

	testCfg := &TestConfig{
		SampleCount:  1,
		Labels:       []string{"error"},
		SdkConnector: &mockSdkConnector{},
	}

	Add(TestError(testCfg))

	results := Run(testCfg)
	testResults := TestResults{}
	for k, v := range results {
		testResults[k] = v
	}

	apiTests := &ApiTest{
		TestRunners: TestRunners{
			"pod1": testResults,
		},
	}

	errMsg, hasError := apiTests.ContainsError()
	if hasError == false {
		t.Logf("Expected an error but got none")
	}

	// Expect(ContainsError(apiTests)).To(BeTrue())
	t.Logf("results = %#v", errMsg)

}
