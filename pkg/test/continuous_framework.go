package test

import (
	"time"

	"github.com/golang/glog"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

type ContinuousTestConfig struct {
	// channel to recieve results on
	resultsCh chan<- Result
	// labels of the tests to run.
	Labels []string
}

func (t *TestSuite) RunContinuous(cfg ContinuousTestConfig) {
	testCases := make(map[string]TestCase)

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

	time.Sleep(50 * time.Millisecond)

	// Run "BeforeSuite" callbacks.
	for _, callback := range t.beforeAllCallbacks {
		err := callback()
		if err != nil {
			glog.Fatalf("Failed to run 'before suite' callbacks: %v", err)
			return
		}
	}

	// This goroutine will run the tests cases in seperate goroutines
	// and wait for them to finish before closing the channel, informing the
	// main goroutine that the tests have finished.
	go func(ch chan<- Result) {
		for name, test := range testCases {
			glog.Infof("  -- running %s\n", name)
			testFunc := test.TestFunc
			setup := test.Setup
			teardown := test.Teardown

			go func(name string, test TestCase) {
				var err error
				var response *sdk.Response
				var latency int64

				if setup != nil {
					if err = setup(test.State); err != nil {
						glog.Errorf("Failed to run setup for test '%s': %v", name, err)
						return
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
					return
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

			}(name, test)
		}
	}(cfg.resultsCh)

}
