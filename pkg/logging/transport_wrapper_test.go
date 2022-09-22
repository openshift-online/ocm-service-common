package logging

// This file contains tests for the logging transport wrapper.

import (
	"context"
	"errors"
	"net/http"

	sdklogging "github.com/openshift-online/ocm-sdk-go/logging"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transport wrapper", func() {
	It("Doesn't panic if there is no response", func() {
		// Create a context:
		ctx := context.Background()

		// Create a logger:
		logger, err := sdklogging.NewStdLoggerBuilder().
			Streams(GinkgoWriter, GinkgoWriter).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Wrap a transport that always fails:
		wrapper, err := NewTransportWrapper().
			Logger(logger).
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())
		transport := wrapper.Wrap(&alwaysFailRoundTripper{})

		// Check that the transport doesn't panic:
		request, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(request).ToNot(BeNil())
		response, err := transport.RoundTrip(request)
		Expect(err).To(HaveOccurred())
		Expect(response).To(BeNil())
	})
})

// alwaysFailRoundTripper is a round tripper that always returns an error and no response.
type alwaysFailRoundTripper struct {
}

// RoundTrip is the implementation of the round tripper interface.
func (t *alwaysFailRoundTripper) RoundTrip(request *http.Request) (response *http.Response, err error) {
	err = errors.New("something failed")
	return
}
