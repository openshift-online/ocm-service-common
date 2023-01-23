package jira

import (
	"io"
	"net/http"

	"github.com/andygrunwald/go-jira"
	errors "github.com/zgalor/weberr"

	"gitlab.cee.redhat.com/service/ocm-common/utils"
)

// This code is based on https://github.com/andygrunwald/go-jira
// Please refer to the URL above to learn about all options that can be used
// to open a Jira ticket via this code.

type Client struct {
	jiraClient *jira.Client
}

func NewClient(user, pass, url string) (*Client, error) {
	err := validateParams(
		Parameter{"jira_user", user},
		Parameter{"jira_pass", pass},
		Parameter{"jira_url", url})
	if err != nil {
		return nil, err
	}
	transport := jira.BasicAuthTransport{
		Username: user,
		Password: pass,
	}
	return newClient(func() *http.Client {
		return transport.Client()
	}, url)
}

func NewClientWithToken(token, url string) (*Client, error) {
	err := validateParams(
		Parameter{"jira_token", token},
		Parameter{"jira_url", url})
	if err != nil {
		return nil, err
	}
	transport := TokenTransport{token: token}
	return newClient(func() *http.Client {
		return transport.Client()
	}, url)
}

type ClientProvider = func() *http.Client

func newClient(clientProvider ClientProvider, url string) (*Client, error) {
	jiraClient, err := jira.NewClient(clientProvider(), url)
	if err != nil {
		return nil, err
	}
	return &Client{jiraClient: jiraClient}, nil
}

type Parameter struct {
	Name  string
	Value string
}

func validateParams(params ...Parameter) error {
	rules := make([]utils.ValidateRule, 0)
	for _, param := range params {
		value := param.Value
		rule := utils.ValidateStringFieldNotEmpty(&value, param.Name)
		rules = append(rules, rule)
	}
	if err := utils.Validate(rules); err != nil {
		return err
	}
	return nil
}

func (c *Client) validateFieldsConfig(fieldsConfig *FieldsConfiguration) error {
	rules := []utils.ValidateRule{
		utils.ValidateNilObject(fieldsConfig, "Fields configuration"),
		utils.ValidateStringFieldNotEmpty(fieldsConfig.Summary, "summary"),
		utils.ValidateStringFieldNotEmpty(fieldsConfig.Reporter, "reporter"),
		utils.ValidateStringFieldNotEmpty(fieldsConfig.IssueType, "issue_type"),
		utils.ValidateStringFieldNotEmpty(fieldsConfig.Project, "project"),
	}
	if err := utils.Validate(rules); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetProjectList() (*jira.ProjectList, *jira.Response, error) {
	return c.jiraClient.Project.GetList()
}

func (c *Client) CreateIssue(fieldsConfig *FieldsConfiguration) (issue *jira.Issue, err error) {
	err = c.validateFieldsConfig(fieldsConfig)
	if err != nil {
		return nil, err
	}

	newIssue := jira.Issue{
		Fields: &jira.IssueFields{
			Summary: *fieldsConfig.Summary,
			Reporter: &jira.User{
				Name: *fieldsConfig.Reporter,
			},
			Type: jira.IssueType{
				Name: *fieldsConfig.IssueType,
			},
			Project: jira.Project{
				Key: *fieldsConfig.Project,
			},
		},
	}

	c.addIssueFields(newIssue, fieldsConfig)

	issue, _, err = c.jiraClient.Issue.Create(&newIssue)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

func (c *Client) addIssueFields(newIssue jira.Issue, fieldsConfig *FieldsConfiguration) {
	// assignee
	if fieldsConfig != nil {
		// assignee
		if fieldsConfig.Assignee != nil && *fieldsConfig.Assignee != "" {
			newIssue.Fields.Assignee = &jira.User{
				Name: *fieldsConfig.Assignee,
			}
		}

		// description
		if fieldsConfig.Description != nil && *fieldsConfig.Description != "" {
			newIssue.Fields.Description = *fieldsConfig.Description
		}

		// label/s
		for _, label := range fieldsConfig.Labels {
			if label != nil && *label != "" {
				newIssue.Fields.Labels = append(newIssue.Fields.Labels, *label)
			}
		}

		// componenet/s
		for _, component := range fieldsConfig.Components {
			if component != nil && *component != "" {
				issueComponent := &jira.Component{
					Name: *component,
				}
				newIssue.Fields.Components = append(newIssue.Fields.Components, issueComponent)
			}
		}

		// custom unknown fields
		var unknowns map[string]interface{}
		for unknownKey, unknownValue := range fieldsConfig.Unknowns {
			knownKey := getUnknownCustomField(unknownKey)
			if knownKey == "" {
				continue
			}
			unknowns[knownKey] = unknownValue
		}
		if len(unknowns) != 0 {
			newIssue.Fields.Unknowns = unknowns
		}
	}
}

func (c *Client) PostAttachment(issueID *string, r io.Reader, name string) (attachment *[]jira.Attachment, err error) {
	if r == nil || issueID == nil {
		return nil, errors.BadRequest.Errorf("Cannot post Jira issue attachment. Missing information")
	}
	createdAttachment, _, err := c.jiraClient.Issue.PostAttachment(*issueID, r, name)
	if err != nil {
		return nil, err
	}
	return createdAttachment, nil
}

func (c *Client) GetAllIssues(searchString string, maxResults int) ([]jira.Issue, error) {
	last := 0
	issues := make([]jira.Issue, 0)
	for {
		opt := &jira.SearchOptions{
			MaxResults: maxResults, // Max results can go up to 1000
			StartAt:    last,
		}

		chunk, resp, err := c.jiraClient.Issue.Search(searchString, opt)
		if err != nil {
			return nil, err
		}

		total := resp.Total
		issues = append(issues, chunk...)
		last = resp.StartAt + len(chunk)
		if last >= total {
			return issues, nil
		}
	}
}
