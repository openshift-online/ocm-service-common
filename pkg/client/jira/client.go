package jira

import (
	"reflect"
	"strings"
	"io"
	"fmt"
	
	"github.com/andygrunwald/go-jira"
	errors "github.com/zgalor/weberr"
)

// This code is based on https://github.com/andygrunwald/go-jira
// Please refer to the URL above to learn about all options that can be used
// to open a Jira ticket via this code.

type Client struct {
	jiraClient   *jira.Client
}

func NewClient(user, pass, url string) (*Client, error) {
	client := &Client{}

	authTransport := jira.BasicAuthTransport{
		Username: user,
		Password: pass,
	}

	jiraClient, err := jira.NewClient(authTransport.Client(), url)
	if err != nil {
		return nil, err
	}
	client.jiraClient = jiraClient
	return client, nil
}

func (c *Client) CreateIssue(fieldsConfig *FieldsConfiguration) (issue *jira.Issue, err error) {
	if fieldsConfig == nil {
		return nil, errors.BadRequest.Errorf("Jira client configuration is nil")
	}

	rules := []validateRule{
		validateNilField(fieldsConfig.Summary, "Summary"),
		validateStringParameterNotEmpty(fieldsConfig.Summary, "Summary"),
		validateNilField(fieldsConfig.Reporter, "Reporter"),
		validateStringParameterNotEmpty(fieldsConfig.Reporter, "Reporter"),
		validateNilField(fieldsConfig.IssueType, "IssueType"),
		validateStringParameterNotEmpty(fieldsConfig.IssueType, "IssueType"),
		validateNilField(fieldsConfig.Project, "Project"),
		validateStringParameterNotEmpty(fieldsConfig.Project, "Project"),
	}
	if err := validate(rules); err != nil {
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
	if !reflect.ValueOf(fieldsConfig.Assignee).IsNil() && *fieldsConfig.Assignee != "" {
		newIssue.Fields.Assignee = &jira.User{
			Name: *fieldsConfig.Assignee,
		}
	}
	// description
	if !reflect.ValueOf(fieldsConfig.Description).IsNil() && *fieldsConfig.Description != "" {
		newIssue.Fields.Description = *fieldsConfig.Description
	}

	// label
	if !reflect.ValueOf(fieldsConfig.Label).IsNil() && *fieldsConfig.Label != "" {
		newIssue.Fields.Labels = append(newIssue.Fields.Labels, fmt.Sprintf("%s", *fieldsConfig.Label))
	}
}

func (c *Client) PostAttachment(r io.Reader, issueID *string) (attachment *[]jira.Attachment, err error) {
	if r == nil || issueID == nil {
		return nil, errors.BadRequest.Errorf("Cannot post Jira issue attachment. Missing information")
	}
	createdAttachment, _, err := c.jiraClient.Issue.PostAttachment(*issueID, r, "clusterResources" )
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

		chunk, resp, err :=  c.jiraClient.Issue.Search(searchString, opt)
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

type validateRule func() error

func validate(rules []validateRule) error {
	for _, rule := range rules {
		if err := rule(); err != nil {
			return err
		}
	}
	return nil
}

func validateNilField(field interface{}, name string) validateRule {
	return func() error {
		if reflect.ValueOf(field).IsNil() {
			return errors.BadRequest.UserErrorf("Missing field '%s'", name)
		}
		return nil
	}
}

func validateStringParameterNotEmpty(param *string, name string) validateRule {
	return func() error {
		if strings.ReplaceAll(*param, " ", "") == "" {
			return errors.BadRequest.UserErrorf("Missing field '%s'", name)
		}
		return nil
	}
}