package jira

type FieldsConfiguration struct {
	Summary     *string
	Description *string
	Project     *string
	Reporter    *string
	Assignee    *string
	IssueType   *string
	Priority    *string
	Labels      []*string
	Components  []*string
	Unknowns    map[string]interface{}
}
