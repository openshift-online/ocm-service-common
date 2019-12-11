package test

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/golang/glog"
)

const (
	DONE    string = "done"
	TESTING string = "testing"

	DEFAULT_SECRET = "apicredentials"
	DEFAUL_URL     = "https://api.stage.openshift.com"
)

// TestRunners have keys that are concurrent nodes processing these tests
type TestRunners map[string]TestResults

// TestResults are the names of distinct test scenario along with a list of elapsed times
// for each of the iterations performed of that scenario.
type TestResults map[string]TestResult

// TestResult is a list of elapsed times as a string
type TestResult []string

func (t *TestRunners) JSON() string {
	b, err := json.Marshal(t)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (t *TestResults) JSON() string {
	b, err := json.Marshal(t)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func FromMap(m map[string][]string) TestResults {
	tr := TestResults{}
	for k, v := range m {
		tr[k] = v
	}
	return tr
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

func (a *ApiTest) ContainsError() (string, bool) {
	for podName, testRunner := range a.TestRunners {
		for testName, results := range testRunner {
			for _, result := range results {
				_, err := time.ParseDuration(result)
				if err != nil {
					glog.Errorf("TestRunner %s and test %s contains error: %s", podName, testName, result)
					return result, true
				}
			}
		}
	}
	return "", false
}