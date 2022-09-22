package logging

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openshift-online/ocm-sdk-go/logging"
)

const opIDHeader = "X-Operation-ID"

type Transport struct {
	Logger  logging.Logger
	Wrapped http.RoundTripper
	// Bodies will be logged iff request path ("/...") starts with ANY of the prefixes.
	// A single "" prefix means always log body; and empty array means never log.
	LogRequestBodyPrefixes  []string
	LogResponseBodyPrefixes []string
	// Bodies won't be logged if the path ("/...") starts with ANY of the prefixes exclusions.
	LogRequestBodyPrefixExclusions  []string
	LogResponseBodyPrefixExclusions []string
}

func (t *Transport) RoundTrip(request *http.Request) (response *http.Response, err error) {
	msg := fmt.Sprintf("Sending %s %s", request.Method, request.URL.String())
	shouldLog := t.shouldLog(request.URL.Path, t.LogRequestBodyPrefixes, t.LogRequestBodyPrefixExclusions)
	if request.Body != nil && shouldLog {
		payload, err := t.getPayload(request.Context(), request.Body)
		if err != nil {
			return nil, err
		}
		msg = fmt.Sprintf("%s: %s", msg, payload)
		request.Body = io.NopCloser(strings.NewReader(payload))
	}
	t.Logger.Info(request.Context(), "%s", msg)

	response, err = t.Wrapped.RoundTrip(request)
	if err != nil {
		return
	}

	msg = fmt.Sprintf("Got back http %d for %s %s", response.StatusCode, request.Method, request.URL.String())
	opID := response.Header.Get(opIDHeader)
	if opID != "" {
		msg = fmt.Sprintf("%s [op-id=%s]", msg, opID)
	}
	shouldLog = t.shouldLog(request.URL.Path, t.LogResponseBodyPrefixes, t.LogResponseBodyPrefixExclusions)
	if response.Body != nil && shouldLog {
		payload, err := t.getPayload(request.Context(), response.Body)
		if err != nil {
			return nil, err
		}
		msg = t.addPayloadToMessage(msg, &payload)
		response.Body = io.NopCloser(strings.NewReader(payload))
	}
	t.Logger.Info(request.Context(), msg)
	return
}

func (t *Transport) addPayloadToMessage(msg string, payload *string) string {
	const maxPayloadLengthToLog = 10000
	var payloadToLog *string
	if len(*payload) > maxPayloadLengthToLog {
		payloadToLog = t.toPointer(fmt.Sprintf("%s... trimmed - payload length (%d) is too long",
			(*payload)[0:maxPayloadLengthToLog], len(*payload)))
	} else {
		payloadToLog = payload
	}
	return fmt.Sprintf("%s: %s", msg, *payloadToLog)
}

func (t *Transport) toPointer(str string) *string {
	return &str
}

func (t *Transport) shouldLog(path string, includePrefixes []string, excludePrefixes []string) bool {
	for _, prefix := range excludePrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}

	for _, prefix := range includePrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

func (t *Transport) getPayload(ctx context.Context, body io.Reader) (payload string, err error) {
	bytes, err := io.ReadAll(body)
	if err != nil {
		t.Logger.Error(ctx, "failed to read body")
		return
	}
	return string(bytes), nil
}
