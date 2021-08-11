package jira

import (
	"fmt"
	"strings"

	"github.com/andygrunwald/go-jira"
	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/gomega"
)

const (
	testURL   = "http://example.com/api/"
	issueType = "Incident"
	project   = "OHSS"
	reporter  = "ocm.support"
	summary   = "\"OCM cluster in error detected: cluster id '%s'\""
	component = "Red Hat OpenShift Cluster Manager"
	labels    = "no_qe"
)

var _ = Describe("Jira issue", func() {

	It("Reject missing client user", func() {
		_, err := NewClient("", "pass", testURL)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Field 'jira_user' is empty"))
	})

	It("Reject missing client password", func() {
		_, err := NewClient("user", "", testURL)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Field 'jira_pass' is empty"))
	})

	It("Reject missing client url", func() {
		_, err := NewClient("user", "pass", "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Field 'jira_url' is empty"))
	})

	It("Reject missing summary", func() {
		jiraClient, err := NewClient("user", "pass", testURL)
		Expect(err).NotTo(HaveOccurred())

		fieldsConfigurattion := &FieldsConfiguration{
			Project:   GetStringAddress(project),
			Reporter:  GetStringAddress(reporter),
			IssueType: GetStringAddress(issueType),
		}

		err = jiraClient.validateFieldsConfig(fieldsConfigurattion)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing field 'summary'"))
	})

	It("Reject missing project", func() {
		jiraClient, err := NewClient("user", "pass", testURL)
		Expect(err).NotTo(HaveOccurred())

		fieldsConfigurattion := &FieldsConfiguration{
			Summary:   GetStringAddress(fmt.Sprintf(summary, "1234")),
			Reporter:  GetStringAddress(reporter),
			IssueType: GetStringAddress(issueType),
		}

		err = jiraClient.validateFieldsConfig(fieldsConfigurattion)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing field 'project'"))
	})

	It("Reject missing reporter", func() {
		jiraClient, err := NewClient("user", "pass", testURL)
		Expect(err).NotTo(HaveOccurred())

		fieldsConfigurattion := &FieldsConfiguration{
			Summary:   GetStringAddress(fmt.Sprintf(summary, "1234")),
			Project:   GetStringAddress(project),
			IssueType: GetStringAddress(issueType),
		}

		err = jiraClient.validateFieldsConfig(fieldsConfigurattion)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing field 'reporter'"))
	})

	It("Reject missing issue type", func() {
		jiraClient, err := NewClient("user", "pass", testURL)
		Expect(err).NotTo(HaveOccurred())

		fieldsConfigurattion := &FieldsConfiguration{
			Summary:  GetStringAddress(fmt.Sprintf(summary, "1234")),
			Project:  GetStringAddress(project),
			Reporter: GetStringAddress(reporter),
		}

		err = jiraClient.validateFieldsConfig(fieldsConfigurattion)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing field 'issue_type'"))
	})

	It("Accept adding components", func() {
		jiraClient, err := NewClient("user", "pass", testURL)
		Expect(err).NotTo(HaveOccurred())

		var fieldsConfigComponents []*string
		newJiraComponent := string(component)
		fieldsConfigComponents = append(fieldsConfigComponents, &newJiraComponent)

		fieldsConfigurattion := &FieldsConfiguration{
			Summary:    GetStringAddress(fmt.Sprintf(summary, "1234")),
			Project:    GetStringAddress(project),
			Reporter:   GetStringAddress(reporter),
			IssueType:  GetStringAddress(issueType),
			Components: fieldsConfigComponents,
		}
		err = jiraClient.validateFieldsConfig(fieldsConfigurattion)
		Expect(err).ToNot(HaveOccurred())

		newIssue := jira.Issue{
			Fields: &jira.IssueFields{
				Summary: *fieldsConfigurattion.Summary,
				Reporter: &jira.User{
					Name: *fieldsConfigurattion.Reporter,
				},
				Type: jira.IssueType{
					Name: *fieldsConfigurattion.IssueType,
				},
				Project: jira.Project{
					Key: *fieldsConfigurattion.Project,
				},
			},
		}
		jiraClient.addIssueFields(newIssue, fieldsConfigurattion)
		Expect(newIssue.Fields.Components).NotTo(BeEmpty())
		Expect(newIssue.Fields.Components[0].Name).To(ContainSubstring(component))
	})

	It("Accept adding labels", func() {
		jiraClient, err := NewClient("user", "pass", testURL)
		Expect(err).NotTo(HaveOccurred())

		var fieldsConfigLabels []*string
		jiraLabels := strings.Split(labels, ",")
		for _, label := range jiraLabels {
			str := string(label)
			fieldsConfigLabels = append(fieldsConfigLabels, &str)
		}

		fieldsConfigurattion := &FieldsConfiguration{
			Summary:   GetStringAddress(fmt.Sprintf(summary, "1234")),
			Project:   GetStringAddress(project),
			Reporter:  GetStringAddress(reporter),
			IssueType: GetStringAddress(issueType),
			Labels:    fieldsConfigLabels,
		}
		err = jiraClient.validateFieldsConfig(fieldsConfigurattion)
		Expect(err).ToNot(HaveOccurred())

		newIssue := jira.Issue{
			Fields: &jira.IssueFields{
				Summary: *fieldsConfigurattion.Summary,
				Reporter: &jira.User{
					Name: *fieldsConfigurattion.Reporter,
				},
				Type: jira.IssueType{
					Name: *fieldsConfigurattion.IssueType,
				},
				Project: jira.Project{
					Key: *fieldsConfigurattion.Project,
				},
			},
		}
		jiraClient.addIssueFields(newIssue, fieldsConfigurattion)
		Expect(newIssue.Fields.Labels).NotTo(BeEmpty())
		Expect(newIssue.Fields.Labels).To(ConsistOf(labels))
	})
})

func GetStringAddress(str string) *string {
	return &str
}
