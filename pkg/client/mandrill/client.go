package mandrill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"
)

type Client struct {
	httpClient *http.Client

	// Configuration
	Config  *ClientConfiguration
	BaseURL *url.URL

	// Services
	TemplateService MandrillService
}

func NewClient(config *ClientConfiguration) (*Client, error) {
	client := &Client{
		Config: config,
	}

	// Ensure baseURL can be parsed and has a trailing slash
	baseURL := strings.TrimSuffix(config.BaseURL, "/")
	var err error
	client.BaseURL, err = url.Parse(baseURL + "/")
	if err != nil {
		return nil, err
	}

	client.httpClient = &http.Client{
		Timeout: config.Timeout,
	}

	client.TemplateService = &TemplateService{client: client}

	return client, nil
}

func NewClientMock(config *ClientConfiguration) (*Client, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	client.TemplateService = &TemplateServiceMock{client: client}
	return client, nil
}

func (c *Client) newRequest(method string, path string, query map[string]string, body interface{}) (*http.Request, error) {
	var u *url.URL
	rel := &url.URL{Path: path}
	u = c.BaseURL.ResolveReference(rel)
	if query != nil {
		q := rel.Query()
		for k, v := range query {
			q.Add(k, v)
		}
		rel.RawQuery = q.Encode()
	}

	var buf io.ReadWriter
	if body != nil {

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		// Mandrill API key is part of json body in the request
		apiKeyJson := fmt.Sprintf("{\"key\":\"%s\",", c.Config.Key)
		bodyWithKey := strings.Replace(string(bodyBytes), "{", apiKeyJson, 1)
		buf = bytes.NewBuffer([]byte(bodyWithKey))
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *Client) do(req *http.Request, marshalInto interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	err = checkResponse(c, req, resp)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	if marshalInto != nil {
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(marshalInto)
	}

	return resp, err
}

type service struct {
	client *Client
}
