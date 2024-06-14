package test

import (
	"io"
	"sync"
	"time"

	"github.com/golang/glog"

	errors "github.com/zgalor/weberr"

	sdk "github.com/openshift-online/ocm-sdk-go"
)

type TestSuite struct {
	// connection is an OCM API client connection that is shared across all tests
	// Go's http.Client is safe for concurrent use across goroutines
	connection *sdk.Connection

	// map of label/test-name/test-func.
	tests map[string]map[string]TestCase

	// a slice of callbacks to run before the test suite.
	beforeAllCallbacks []TestCallback

	// a slice of callbacks to run After the test suite.
	afterAllCallbacks []TestCallback

	// the longest duration to wait for a request before timing out of a test suite run.
	timeout time.Duration

	// the default account id of the test suite
	defaultAccountID string
}

type TestSuiteSpec struct {
	SdkConnector       SDKConnector
	Tests              map[string]map[string]TestCase
	BeforeAllCallbacks []TestCallback
	AfterAllCallbacks  []TestCallback
	BaseURL            string
	TokenURL           string
	SecretName         string
	ClientId           string
	ClientSecret       string
	Token              string
	DefaultAccountID   string
	Debug              bool
	Timeout            time.Duration
}

type TestConfig struct {
	// The number of times to run each test.
	SampleCount int
	// Tests run are filtered using these labels.
	Labels []string
}

func BuildTestSuite(spec *TestSuiteSpec) (*TestSuite, error) {
	conn, err := spec.SdkConnector.Connect(spec)
	if err != nil {
		return nil, errors.Errorf("Unable to run test framework, connection to SDK must be provided: %v", err)
	}

	// Default timeout to five minutes.
	if spec.Timeout == 0 {
		spec.Timeout = 5 * time.Minute
	}

	return &TestSuite{
		connection:       conn,
		tests:            spec.Tests,
		timeout:          spec.Timeout,
		defaultAccountID: spec.DefaultAccountID,
	}, nil
}

func (t *TestSuite) Connection() *sdk.Connection {
	return t.connection
}

// Run will run all the requests and assertions in the suite that match a given set of labels.
// Each test will be run as many times as provided by SampleCount
func (t *TestSuite) Run(cfg *TestConfig) map[string][]Result {
	testCases := make(map[string]TestCase)
	results := make(map[string][]Result)

	defer func() {
		// Run "AfterSuite" callbacks.
		for _, callback := range t.afterAllCallbacks {
			err := callback()
			if err != nil {
				glog.Fatalf("Failed to run 'after suite' callbacks: %v", err)
			}
		}
	}()

	// Tests can appear in different labels.
	// For that reason we need to "flatten" the two-level mapping.
	// That is, go over each label and collect the test cases to one
	// mapping which maps each name to their respective test.
	// this mapping can then be used to run all tests.
	// cfg.Labels is by default []string{"all"}.
	for _, label := range cfg.Labels {
		if tests, found := t.tests[label]; found {
			for testCaseName, testCase := range tests {
				if _, found := testCases[testCaseName]; !found {
					testCases[testCaseName] = testCase
				}
			}
		}
	}

	testCount := len(testCases)

	// non-blocking buffered channel that receives test run data
	ch := make(chan Result, testCount*cfg.SampleCount)

	time.Sleep(50 * time.Millisecond)

	// Run "BeforeSuite" callbacks.
	for _, callback := range t.beforeAllCallbacks {
		err := callback()
		if err != nil {
			glog.Fatalf("Failed to run 'before suite' callbacks: %v", err)
			return nil
		}
	}

	// This goroutine will run the tests cases in seperate goroutines
	// and wait for them to finish before closing the channel, informing the
	// main goroutine that the tests have finished.
	go func(ch chan<- Result) {
		wg := sync.WaitGroup{}
		for name, test := range testCases {
			glog.Infof("  -- running %s\n", name)
			testFunc := test.TestFunc
			setup := test.Setup
			teardown := test.Teardown

			// Add to wait group.
			wg.Add(1)

			go func(name string, test TestCase) {
				for i := 0; i < cfg.SampleCount; i++ {
					var err error
					var response *sdk.Response
					var latency int64

					if setup != nil {
						if err = setup(test.State); err != nil {
							glog.Errorf("Failed to run setup for test '%s': %v", name, err)
							continue
						}
					}
					start := time.Now()
					response, err = testFunc(test.State)
					latency = time.Since(start).Milliseconds()
					if err != nil {
						// Request failed send results immediatly.
						ch <- NewResult(
							name,
							err,
							latency,
							0,
						)
						continue
					}

					responseSize := len(response.Bytes())

					// Run assertions.
					for _, assertion := range test.ResponseAssertions {
						if err = assertion(response); err != nil {
							ch <- NewResult(
								name,
								err,
								latency,
								responseSize,
							)
							// regardless if several assertions fail we still
							// consider this roundtrip result as a single "error".
							break
						}
					}

					// Send succesful result.
					if err == nil {
						ch <- NewResult(
							name,
							nil,
							latency,
							responseSize,
						)
					}

					if teardown != nil {
						if err = teardown(test.State); err != nil {
							glog.Errorf("Failed to run tear down for test '%s': %v", name, err)
						}
					}
				}
				// Decrement the Wait group.
				wg.Done()
			}(name, test)
		}

		// Wait to close the channel.
		wg.Wait()
		close(ch)
	}(ch)

	// flush the channel.
	// this is a stream of results coming in.
	// we can have multiple results for the same test;
	// depending on the cfg.SampleCount field
	timer := time.NewTimer(t.timeout)
	for {
		select {
		case r, ok := <-ch:
			if ok {
				if _, found := results[r.Name]; !found {
					results[r.Name] = make([]Result, 0)
				}
				results[r.Name] = append(results[r.Name], r)
				glog.Infof("received %#v\n", r)

				// reset the request timeout clock.
				timer.Stop()
				timer.Reset(t.timeout)
			} else {
				return results
			}
		case <-timer.C:
			glog.Errorf("Timed out waiting for a result from the tests cases, returning partial results.")
			return results
		}
	}
}

type IndexFunc func() []string

func (t *TestSuite) AddBeforeSuite(callbacks []TestCallback) {
	if t.beforeAllCallbacks == nil {
		t.beforeAllCallbacks = make([]TestCallback, 0)
	}
	t.beforeAllCallbacks = append(t.beforeAllCallbacks, callbacks...)
}

func (t *TestSuite) AddAfterSuite(callbacks []TestCallback) {
	if t.afterAllCallbacks == nil {
		t.afterAllCallbacks = make([]TestCallback, 0)
	}
	t.afterAllCallbacks = append(t.afterAllCallbacks, callbacks...)
}

func (t *TestSuite) AddTestCases(testCases []*TestCase) {
	for _, tc := range testCases {
		t.Add(tc)
	}
}

func (t *TestSuite) Add(testCase *TestCase) {
	if t.tests == nil {
		t.tests = make(map[string]map[string]TestCase)
	}

	if _, found := t.tests["all"]; !found {
		t.tests["all"] = make(map[string]TestCase)
	}

	if _, ok := t.tests["all"][testCase.Name]; ok {
		glog.Fatalf("TestCase[%s/%s] already exists", "all", testCase.Name)
	}

	// special case where we don't want this test in the "all" bucket
	if testCase.Name != ERRORTEST {
		glog.Infof("Adding test: %s/%s\n", "all", testCase.Name)
		t.tests["all"][testCase.Name] = *testCase
	}

	for _, l := range testCase.Labels {
		if _, found := t.tests[l]; !found {
			t.tests[l] = make(map[string]TestCase)
		}

		if _, ok := t.tests[l][testCase.Name]; ok {
			glog.Fatalf("TestCase[%s/%s] already exists", l, testCase.Name)
		}

		glog.Infof("Adding test: %s/%s\n", l, testCase.Name)
		t.tests[l][testCase.Name] = *testCase
	}
}

func (t *TestSuite) GetDefaultAccountID() string {
	return t.defaultAccountID
}

func NewResult(name string, err error, latency int64, size int) Result {
	return Result{
		Name:    name,
		Error:   StringPtrFromErr(err),
		Latency: latency,
		Size:    size,
	}
}

func StringPtrFromErr(err error) *string {
	if err == nil {
		return nil
	}
	return NewString(err.Error())
}

func NewString(str string) *string {
	if str == "" {
		return nil
	}
	return &str
}

func matchAll(pat, str string) (bool, error) {
	return true, nil
}

type matchStringOnly func(pat, str string) (bool, error)

func (f matchStringOnly) MatchString(pat, str string) (bool, error) { return true, nil }
func (f matchStringOnly) StartCPUProfile(w io.Writer) error {
	return errors.Errorf("testing: StartCPUProfile not implemented")
}
func (f matchStringOnly) StopCPUProfile() {}
func (f matchStringOnly) WriteProfileTo(string, io.Writer, int) error {
	return errors.Errorf("testing: WriteProfileTo not implemented")
}
func (f matchStringOnly) ImportPath() string     { return "" }
func (f matchStringOnly) StartTestLog(io.Writer) {}
func (f matchStringOnly) StopTestLog() error     { return nil }
