package mandrill

import (
	"net/http"
	"time"
)

type ClientConfiguration struct {
	BaseURL    string
	Key        string
	HttpClient *http.Client
	Timeout    time.Duration
}

func NewClientConfig() *ClientConfiguration {
	return &ClientConfiguration{
		BaseURL: "https://mandrillapp.com/api/1.0",
		Key:     "abc123",
		Timeout: 5 * time.Second,
	}
}
