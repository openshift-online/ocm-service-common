package segment

import "net/http"

type ClientConfiguration struct {
	BaseURL    string
	Key        string
	HttpClient *http.Client
	Version    string
}

func NewClientConfig() *ClientConfiguration {
	return &ClientConfiguration{
		BaseURL: "https://api.segment.io",
		Key:     "",
	}
}
