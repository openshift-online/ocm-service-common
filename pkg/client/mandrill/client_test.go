package mandrill

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

const (
	testURL = "http://example.com/api/"
	testKey = "abc123"
)

type testParams struct {
	Name string `json:"name"`
}

func newTestClientConfiguration() *ClientConfiguration {
	return &ClientConfiguration{
		BaseURL: testURL,
		Key:     testKey,
	}
}

func TestNewClient_BadUrl(t *testing.T) {
	config := newTestClientConfiguration()
	// Invalid URL
	config.BaseURL = " https://example_com"

	_, err := NewClient(config)
	if err == nil {
		t.Error("Expected error returned due to invalid base url configuration")
	}
}

func TestNewClient(t *testing.T) {
	config := newTestClientConfiguration()

	c, err := NewClient(config)
	if err != nil {
		t.Error("Expected no error creating client")
	}

	if c.BaseURL.String() != config.BaseURL {
		t.Errorf("Expected client baseURL to be %s, but got %s", config.BaseURL, c.BaseURL)
	}

	params := TemplateParams{TemplateName: "Bart"}
	err = c.TemplateService.TemplateSend(params)

	if err == nil {
		t.Error("Expected 404 error sending request")
	}

	if !strings.Contains(err.Error(), "Mandrill error: 404") {
		t.Error("Expected 404 error contacting Mandrill")
	}

}

func TestNewClientMock(t *testing.T) {
	config := newTestClientConfiguration()

	c, err := NewClientMock(config)
	if err != nil {
		t.Error("Expected no error creating client")
	}

	// Reassign log output to capture mock client output
	var output bytes.Buffer
	log.SetOutput(&output)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	params := TemplateParams{TemplateName: "Bart"}
	c.TemplateService.TemplateSend(params)

	match := "Mandrill Send Template Bart"
	if !strings.Contains(output.String(), match) {
		t.Error("Expected output to contain", match)
	}
}

/* Example using Mandrill template and merge vars
// Draft template: https://mandrillapp.com/templates/code?id=ocm-test-template
func TestNewClientTestKey(t *testing.T) {
	config := newTestClientConfiguration()
	config.BaseURL = baseUrl
	config.Key = key

	c, err := NewClient(config)
	if err != nil {
		t.Error("Expected no error creating client")
	}

	params := c.TemplateService.NewTemplateParams()
	params.TemplateName = "OCM Test Template"
	params.Message.Subject = "OCM Test Template"
	params.Message.FromEmail = "no-reply@openshift.com"
	params.Message.FromName = "No Reply"
	params.Message.To[0].Email = "nobody@example.com"
	params.Message.To[0].Name = "Frank Grimes"
	params.Message.GlobalMergeVars[0].Name = "MYMERGETAG"
	params.Message.GlobalMergeVars[0].Content = "<h1>merge replaced</h1>"
	err = c.TemplateService.TemplateSend(params)

	if err != nil {
		t.Error("Error sending request", err)
	}

}
*/
