package notifications

type ClientConfiguration struct {
	BaseURL                 string
	ProxyURL                string
	KeyFile                 string
	CertFile                string
	Key                     string
	Cert                    string
	EnableMock              bool
	UseRHCSCertAutoRotation bool
}

func NewClientConfig() *ClientConfiguration {
	return &ClientConfiguration{
		BaseURL:                 "https://mtls.internal.cloud.redhat.com/api/notifications-gw/notifications",
		ProxyURL:                "",
		KeyFile:                 "secrets/notifications.key",
		CertFile:                "secrets/notifications.crt",
		EnableMock:              false,
		UseRHCSCertAutoRotation: false,
	}
}
