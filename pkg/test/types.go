package test

import (
	"encoding/json"
	"strings"

	"github.com/golang/glog"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

const (
	DONE    string = "done"
	TESTING string = "testing"

	DEFAULT_SECRET = "apicredentials"
	DEFAUL_URL     = "https://api.stage.openshift.com"
)

// A test result.
type Result struct {
	Name    string
	Error   error
	Latency int64
	Size    int
}

// A series of assertion ran on the response of the request sent.
type ResponseAssertion func(*sdk.Response) error

// a callback to be run before or after the Tests are run.
// see AddBeforeAll and AddAfterAll.
type TestCallback func() error

type TestState map[string]interface{}

type TestCase struct {
	Name               string
	Labels             []string
	TestFunc           func(s TestState) (*sdk.Response, error)
	ResponseAssertions []ResponseAssertion
	Setup              func(s TestState) error
	Teardown           func(s TestState) error
	State              TestState
}

// TestRunners have keys that are concurrent nodes processing these tests
type TestRunners map[string]map[string][]Result

func (t *TestRunners) JSON() string {
	b, err := json.Marshal(t)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (t *Result) JSON() string {
	b, err := json.Marshal(t)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

type ApiTest struct {
	Name        string // 1:1 to ConfigMap.Name
	TestID      string
	Status      string
	StartTime   string
	EndTime     string
	Width       int
	Depth       int
	TestRunners TestRunners // pods running a suite of tests
	BaseURL     string
	SecretName  string
	Labels      []string
}

func (a *ApiTest) LabelsToCSV() string {
	return strings.Trim(strings.Join(a.Labels, ", "), "[]")
}

func (a *ApiTest) HasResults(testRunnerName string) bool {
	testResults, found := a.TestRunners[testRunnerName]
	if !found {
		return false
	}

	return len(testResults) > 0
}

func (a *ApiTest) ContainsError() (error, bool) {
	for podName, nameToResults := range a.TestRunners {
		for _, results := range nameToResults {
			for _, result := range results {
				if result.Error != nil {
					glog.Errorf("TestRunner %s and test %s contains error: %s", podName, result.Error, result.Error)
					return result.Error, true
				}
			}
		}
	}
	return nil, false
}
