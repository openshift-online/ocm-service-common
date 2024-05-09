package test

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sdk "github.com/openshift-online/ocm-sdk-go"
	errors "github.com/zgalor/weberr"
)

// Names of the environment variables:
const (
	tokenEnv        = "UHC_TOKEN"
	clientIdEnv     = "AMS_CLIENT_ID"
	clientSecretEnv = "AMS_CLIENT_SECRET"
)

type SDKConnector interface {
	Connect(spec *TestSuiteSpec) (*sdk.Connection, error)
}

type sdkConnector struct{}

type mockSdkConnector struct{}

type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	fmt.Println("roundtrip")
	return &http.Response{
		StatusCode: http.StatusOK,
	}, nil
}

func (c *mockSdkConnector) Connect(spec *TestSuiteSpec) (*sdk.Connection, error) {
	return sdk.NewConnectionBuilder().TransportWrapper(func(wrapped http.RoundTripper) http.RoundTripper {
		fmt.Println("swapping round tripper")
		return new(mockRoundTripper)
	}).Client("foo", "bar").Build()
}

/*
*
Connect creates a connection to the environment specified by the AOC_USER, AOC_PASSWORD and
AOC_DOMAIN environment variables.

If a refresh / offline token is provided, it is used with the default `cloud-services` client.
If a client and secret is provided then that alone is used.
*/
func (c *sdkConnector) Connect(spec *TestSuiteSpec) (*sdk.Connection, error) {
	t := &testing.T{}
	RegisterTestingT(t)

	// Create a logger:
	logger, err := sdk.NewStdLoggerBuilder().
		Streams(GinkgoWriter, GinkgoWriter).
		Debug(spec.Debug).
		Build()

	if err != nil {
		return nil, err
	}

	builder := sdk.NewConnectionBuilder().
		Logger(logger).
		URL(spec.BaseURL)

	if spec.TokenURL != "" {
		builder = builder.TokenURL(spec.TokenURL)
	}

	// If we don't have anything configured specifically for this test, attempt to rectify from the env
	if spec.Token == "" && spec.ClientId == "" && spec.ClientSecret == "" {
		spec.Token = os.Getenv(tokenEnv)
		spec.ClientId = os.Getenv(clientIdEnv)
		spec.ClientSecret = os.Getenv(clientSecretEnv)
	}

	if spec.Token != "" {
		builder = builder.Tokens(spec.Token)
		glog.Infof("Connecting to uhc sdk with token with last 8 chars: %s", spec.Token[len(spec.Token)-8:])
	} else if spec.ClientId != "" && spec.ClientSecret != "" {
		builder = builder.Client(spec.ClientId, spec.ClientSecret)
		glog.Infof("Connecting to uhc sdk with client/secret with clientId: %s", spec.ClientId)
	} else {
		return nil, errors.Errorf("No token or client/secret found to connect to uhc sdk")
	}

	// Create the connection:
	connection, err := builder.Build()
	if err != nil {
		return nil, err
	}

	accountId := GetAccountID(spec)
	glog.Infof("Using account id %s against BaseURL %s", accountId, spec.BaseURL)

	return connection, nil
}

const (
	PERFORMANCE string = "performance"
	REGRESSION  string = "regression"
)

func NewTestConfig() *TestConfig {
	return &TestConfig{
		SampleCount: 1,
		Labels:      []string{"all"},
	}
}

func NewTestSuiteSpec() *TestSuiteSpec {
	return &TestSuiteSpec{
		BaseURL:          "https://api.stage.openshift.com",
		SecretName:       "stage-creds",
		SdkConnector:     &sdkConnector{},
		DefaultAccountID: "No default account ID set, or unknown environment. Please set in helpers.go",
		Timeout:          5 * time.Minute,
	}
}

func NewMockTestSuiteSpec(mockURL string, mockTokenURL string) *TestSuiteSpec {
	return &TestSuiteSpec{
		BaseURL:          mockURL,
		TokenURL:         mockTokenURL,
		SecretName:       "stage-creds",
		SdkConnector:     &sdkConnector{},
		DefaultAccountID: "No default account ID set, or unknown environment. Please set in helpers.go",
		Timeout:          5 * time.Minute,
		ClientId:         "mock-client",
		ClientSecret:     "mock-secret",
	}
}

func GetEnvironment(url string) string {
	switch url {
	case "https://api.stage.openshift.com":
		return "stage"
	case "https://api.openshift.com":
		return "prod"
	case "https://api-integration.6943.hive-integration.openshiftapps.com":
		return "int"
	default:
		return "dev"
	}
}

func GetAccountID(spec *TestSuiteSpec) string {
	prodID := "1MpGILXFZUlZuLldwGohxGaKxmW"
	stageID := "1Mpeh6PlQVyIJtC1ebJ6GOTx5Pq"
	intID := "1Nk17U3WwVgLWUHeGlNGcpcMasI"

	switch spec.BaseURL {
	case "https://api.stage.openshift.com":
		return stageID
	case "https://api.openshift.com":
		return prodID
	case "https://api-integration.6943.hive-integration.openshiftapps.com":
		return intID
	default:
		return spec.DefaultAccountID
	}
}

const ERRORTEST = "error"

// TestError is a special purpose scenario for testing sentry integration
func TestError(suite *TestSuite) *TestCase {
	return &TestCase{
		Name: ERRORTEST,
		Labels: []string{
			"error",
		},
		TestFunc: func(s TestState) (*sdk.Response, error) {
			return suite.Connection().Get().Path("/api/clusters_mgmt/v1/").Send()
		},
	}
}

func AssertResponseStatusOK() ResponseAssertion {
	return func(r *sdk.Response) error {
		if r.Status() != http.StatusOK {
			return errors.Errorf("Expected response code %d, but found: %d",
				http.StatusOK, r.Status())
		}
		return nil
	}
}
