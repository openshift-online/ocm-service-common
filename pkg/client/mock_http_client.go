package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

const (
	MockTokenURL     = "http://mock.token.url"
	MockClientID     = "id"
	MockClientSecret = "secret"
	MockAccessToken  = "eyJ0eXAiOiJKV1QiLA0KICJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJqb2UiLA0KICJleHAiOjEzMDA4MTkzODAsDQogImh0dHA6Ly9leGFtcGxlLmNvbS9pc19yb290Ijp0cnVlfQ.dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	MockRefreshToken = "eyJ0eXAiOiJKV1QiLA0KICJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJqb2UiLA0KICJleHAiOjEzMDA4MTkzODAsDQogImh0dHA6Ly9leGFtcGxlLmNvbS9pc19yb290Ijp0cnVlfQ.dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
)

var MockJsonHeader = http.Header{"Content-Type": {"application/json"}}

type mockRoundTripper struct {
	code        int
	header      http.Header
	body        string
	interceptor *testInterceptor
}

func (m *mockRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	code := m.code
	body := m.body
	header := m.header
	m.interceptor.mostRecentRequest = request
	// bypass sso
	if "http://"+request.Host == MockTokenURL {
		header = MockJsonHeader
		code = 200
		body = fmt.Sprintf(`{"token_type":"bearer","access_token":"%s","refresh_token":"%s"}`, MockAccessToken, MockRefreshToken)
	}
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     header,
	}, nil
}

type testInterceptor struct {
	mostRecentRequest *http.Request
}

func (m *testInterceptor) GetMostRecentRequest() *http.Request {
	return m.mostRecentRequest
}

// NewMockHttpClient returns :
// - a mock http.Client that returns the mock response;
// - a mock interceptor that stores values to be examined in tests.
// param:
// code - response status code;
// header - response header;
// body - response body.
func NewMockHttpClient(code int, header http.Header, body string) (*http.Client, *testInterceptor) {
	interceptor := &testInterceptor{}
	client := &http.Client{Transport: &mockRoundTripper{code, header, body, interceptor}}
	return client, interceptor
}
