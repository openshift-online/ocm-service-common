package test

import (
"errors"
"fmt"
"io"
"testing"
"time"

. "github.com/onsi/gomega"
 "github.com/openshift-online/ocm-sdk-go"
)

// map of label/test-name/test-func
var Tests map[string]map[string]TestFunc

type TestFunc func(t *testing.T)

type TestCase struct {
	Name     string
	Labels   []string
	TestFunc TestFunc
}

// connection is an OCM API client connection that is shared across all tests
// Go's http.Client is safe for concurrent use across goroutines
var Connection *sdk.Connection

// Run will run all tests in the suite.
func Run(cfg *TestConfig) map[string][]string {

	c, err := cfg.SdkConnector.Connect(cfg)
	Connection = c

	fmt.Errorf("Error connecting to sdk: %s", err)

	results := map[string][]string{}

	type result struct {
		name    string
		elapsed []string
	}

	// how many tests are going to run?
	// each label has an indeterminate number of tests.
	// TODO: better way to count these??
	testCount := 0
	for _, label := range cfg.Labels {
		if tests, found := Tests[label]; found {
			testCount = testCount + len(tests)
		}
	}

	// non-blocking buffered channel that receives test run data
	ch := make(chan result, testCount)

	go func() {
		time.Sleep(50 * time.Millisecond)
		for _, label := range cfg.Labels {
			for name, test := range Tests[label] {
				fmt.Printf("  -- running %s/%s\n", label, name)
				testFn := test
				func(n string) {

					results := []string{}
					for i := 0; i < cfg.SampleCount; i++ {
						start := time.Now()

						m := testing.MainStart(
							matchStringOnly(matchAll),
							[]testing.InternalTest{
								{
									Name: n,
									F: func(t *testing.T) {
										RegisterTestingT(t)
										testFn(t)
									},
								},
							},
							[]testing.InternalBenchmark{},
							[]testing.InternalExample{})

						status := m.Run()
						elapsed := time.Since(start).String()

						if status > 0 {
							elapsed = n + " failed in " + elapsed
						}

						results = append(results, elapsed)
					}

					// sends results from this goroutine through the channel back to the main process
					ch <- result{n, results}
				}(name)
			}
		}
	}()

	for i := 0; i < testCount; i++ {
		// blocks until receiving something from the channel
		// will block testCount times in this loop until all tests have reported back
		r := <-ch
		results[r.name] = r.elapsed
		fmt.Printf("received %#v\n", r)
	}

	return results
}

func init() {
	Tests = make(map[string]map[string]TestFunc)
}

type IndexFunc func() []string

func AddTestCases(testCases []*TestCase) {
	for _, tc := range testCases {
		Add(tc)
	}
}

func Add(testCase *TestCase) {
	if _, found := Tests["all"]; !found {
		Tests["all"] = make(map[string]TestFunc)
	}

	if _, ok := Tests["all"][testCase.Name]; ok {
		panic(fmt.Sprintf("TestCase[%s/%s] already exists", "all", testCase.Name))
	}

	// special case where we don't want this test in the "all" bucket
	if testCase.Name != ERRORTEST {
		fmt.Printf("Adding test: %s/%s\n", "all", testCase.Name)
		Tests["all"][testCase.Name] = testCase.TestFunc
	}

	for _, l := range testCase.Labels {
		if _, found := Tests[l]; !found {
			Tests[l] = make(map[string]TestFunc)
		}
	}

	for _, l := range testCase.Labels {
		if _, ok := Tests[l][testCase.Name]; ok {
			panic(fmt.Sprintf("TestCase[%s/%s] already exists", l, testCase.Name))
		}
		fmt.Printf("Adding test: %s/%s\n", l, testCase.Name)
		Tests[l][testCase.Name] = testCase.TestFunc
	}
}

func matchAll(pat, str string) (bool, error) {
	return true, nil
}

type matchStringOnly func(pat, str string) (bool, error)

func (f matchStringOnly) MatchString(pat, str string) (bool, error) { return true, nil }
func (f matchStringOnly) StartCPUProfile(w io.Writer) error {
	return errors.New("testing: StartCPUProfile not implemented")
}
func (f matchStringOnly) StopCPUProfile() {}
func (f matchStringOnly) WriteProfileTo(string, io.Writer, int) error {
	return errors.New("testing: WriteProfileTo not implemented")
}
func (f matchStringOnly) ImportPath() string     { return "" }
func (f matchStringOnly) StartTestLog(io.Writer) {}
func (f matchStringOnly) StopTestLog() error     { return nil }
