package test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift-online/ocm-sdk-go"
)

// Names of the environment variables:
const (
	tokenEnv        = "UHC_TOKEN"
	clientIdEnv     = "AMS_CLIENT_ID"
	clientSecretEnv = "AMS_CLIENT_SECRET"
)

type SdkConnector interface {
	Connect(cfg *TestConfig) *sdk.Connection
}

type sdkConnector struct{}

type mockSdkConnector struct{}

func (c *mockSdkConnector) Connect(cfg *TestConfig) *sdk.Connection {
	return &sdk.Connection{}
}

/**
Connect creates a connection to the environment specified by the AOC_USER, AOC_PASSWORD and
AOC_DOMAIN environment variables.

If a refresh / offline token is provided, it is used with the default `cloud-services` client.
If a client and secret is provided then that alone is used.
*/
func (c *sdkConnector) Connect(cfg *TestConfig) *sdk.Connection {
	t := &testing.T{}
	RegisterTestingT(t)

	// Create a logger:
	logger, err := sdk.NewStdLoggerBuilder().
		Streams(GinkgoWriter, GinkgoWriter).
		Debug(true).
		Build()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	builder := sdk.NewConnectionBuilder().
		Logger(logger).
		URL(cfg.BaseURL)

	// If we don't have anything configured specifically for this test, attempt to rectify from the env
	if cfg.Token == "" && cfg.ClientId == "" && cfg.ClientSecret == ""{
		cfg.Token = os.Getenv(tokenEnv)
		cfg.ClientId = os.Getenv(clientIdEnv)
		cfg.ClientSecret = os.Getenv(clientSecretEnv)
	}

	if cfg.Token != "" {
		builder = builder.Tokens(cfg.Token)
	} else if cfg.ClientId != "" && cfg.ClientSecret != "" {
		builder = builder.Client(cfg.ClientId, cfg.ClientSecret)
	}

	// Create the connection:
	connection, err := builder.Build()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return connection
}

const (
	PERFORMANCE string = "performance"
	REGRESSION  string = "regression"
)

type TestConfig struct {
	SampleCount  int
	BaseURL      string
	SecretName   string
	Labels       []string
	SdkConnector SdkConnector
	ClientId     string
	ClientSecret string
	Token        string
}

func NewTestConfig() *TestConfig {
	return &TestConfig{
		SampleCount:  1,
		BaseURL:      "https://api.stage.openshift.com",
		SecretName:   "stage-creds",
		Labels:       []string{"all"},
		SdkConnector: &sdkConnector{},
	}
}

func GetAccountID(cfg *TestConfig) string {
	prodID := "1MpGILXFZUlZuLldwGohxGaKxmW"
	stageID := "1Mpeh6PlQVyIJtC1ebJ6GOTx5Pq"

	switch cfg.BaseURL {
	case "https://api.stage.openshift.com":
		return stageID
	case "https://api.openshift.com":
		return prodID
	default:
		return "unknown environment -- please set in helpers.go"
	}
}

const ERRORTEST = "error"

// TestError is a special purpose scenario for testing sentry integration
func TestError(cfg *TestConfig) *TestCase {
	return &TestCase{
		Name: ERRORTEST,
		Labels: []string{
			"error",
		},
		TestFunc: func(t *testing.T) {
			t.Logf("t.logging ... error!")
			Expect(false).To(BeTrue(), "this is a test")
		},
	}
}
