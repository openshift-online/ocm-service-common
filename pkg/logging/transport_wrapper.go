// This file contains the implementations of the round tripper that writes the details of the
// requests to the log.

package logging

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/openshift-online/async-routine/opid"
	sdklogging "github.com/openshift-online/ocm-sdk-go/logging"
)

// ContextKey is the type of keys used to store operation identifiers in contexts.
type ContextKey int

// TransportWrapperBuilder contains the data and logic needed to create logging transport wrappers.
type TransportWrapperBuilder struct {
	logger sdklogging.Logger
}

// TransportWrapper knows how to wrap an HTTP round tripper with another that writes to the log
// details of the requests and responses.
type TransportWrapper struct {
	logger sdklogging.Logger
}

// roundTripper is an implementation of the http.RoundTripper interface that wrapps another round
// tripper and writes to the log details of the requests and responses.
type roundTripper struct {
	logger sdklogging.Logger
	next   http.RoundTripper
}

// responseBody is an implementation of the io.ReadCloser interface that allows us to capture the
// details of the response.
type responseBody struct {
	ctx    context.Context
	logger sdklogging.Logger
	method string
	url    string
	start  time.Time
	length int
	next   io.ReadCloser
}

// NewTransportWrapper creates a builder that can then be used to configure and create a logging
// transport wrapper.
func NewTransportWrapper() *TransportWrapperBuilder {
	return &TransportWrapperBuilder{}
}

// Logger sets the logger that will be used by the created round trippers to write details of the
// requests and responses to the log.
func (b *TransportWrapperBuilder) Logger(value sdklogging.Logger) *TransportWrapperBuilder {
	b.logger = value
	return b
}

// Build uses the data stored in the builder to create a new wrapper.
func (b *TransportWrapperBuilder) Build(ctx context.Context) (result *TransportWrapper, err error) {
	// Check parameters:
	if b.logger == nil {
		err = fmt.Errorf("logger is mandatory")
		return
	}

	// Create and populate the object:
	result = &TransportWrapper{
		logger: b.logger,
	}

	return
}

// Wrap takes an HTTP round tripper and wraps it with another that writes to the log details of the
// requests and responses.
func (w *TransportWrapper) Wrap(next http.RoundTripper) http.RoundTripper {
	return &roundTripper{
		logger: w.logger,
		next:   next,
	}
}

// RoundTrip is the implementation of the http.RoundTripper interface.
func (r *roundTripper) RoundTrip(request *http.Request) (response *http.Response, err error) {
	// Get the context:
	ctx := request.Context()

	// Skip the requests for the URLs that are too noisy or not interesting:
	method := request.Method
	url := request.URL.String()
	if skipRE.MatchString(url) {
		response, err = r.next.RoundTrip(request)
		return
	}

	// Ensure that the context has an operation identifier attached, as otherwise it is
	// difficult to associate the log messages containing the details of a request with the log
	// message containing the details for the response.
	ctx = opid.WithOpId(ctx)
	request = request.WithContext(ctx)

	// Write the request details to the log:
	r.logger.Info(
		ctx,
		"Sending '%s %s'",
		method, url,
	)

	// Send the request and wait for the response headers.
	start := time.Now().UTC()
	response, err = r.next.RoundTrip(request)
	if err != nil {
		return
	}
	status := response.StatusCode
	duration := time.Since(start)
	r.logger.Info(
		ctx,
		"Received %d response for '%s %s' after %s",
		status, method, url, duration,
	)

	// We can't know the actual details of the response till the body has been completely read.
	// In particular, we can't know what is the response time or length. To do so we need to
	// replace the response body with one that allows us to get that information.
	response.Body = &responseBody{
		ctx:    ctx,
		logger: r.logger,
		method: method,
		url:    url,
		start:  start,
		next:   response.Body,
	}

	return
}

// Read is part of the implementation of the io.ReadCloser interface.
func (b *responseBody) Read(p []byte) (n int, err error) {
	n, err = b.next.Read(p)
	b.length += n
	return
}

// Read is part of the implementation of the io.ReadCloser interface.
func (b *responseBody) Close() error {
	// When the response body is closed we can write to the log the actual response details,
	// specially the response length:
	duration := time.Since(b.start)
	b.logger.Info(
		b.ctx,
		"Received %d bytes for '%s %s' after %s",
		b.length, b.method, b.url, duration,
	)

	// Close the wrapped body:
	return b.next.Close()
}

// skipRE is the regular expression that will be used to discard URLs that are too noisy or not
// interesting. Kubernetes API leader election, for example, produces requests every two seconds and
// it isn't useful to have that in the log.
var skipRE = regexp.MustCompile("/namespaces/.*-leadership/")
