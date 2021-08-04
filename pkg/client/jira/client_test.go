package jira

import (
	"fmt"
	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/gomega"
)

const (
	testURL   = "http://example.com/api/"
	issueType = "Bug"
	project   = "OHSS"
	reporter  = "ocm.support"
	summary   = "\"OCM cluster in error detected: cluster id '%s'\""
)

var _ = Describe("Jira issue", func() {

	It("Missing client user", func() {
		_, err := NewClient("", "pass", testURL)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Field 'jira_user' is empty"))
	})

	It("Missing client password", func() {
		_, err := NewClient("user", "", testURL)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Field 'jira_pass' is empty"))
	})

	It("Missing client url", func() {
		_, err := NewClient("user", "pass", "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Field 'jira_url' is empty"))
	})

	It("Missing summary", func() {
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

	It("Missing project", func() {
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

	It("Missing reporter", func() {
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

	It("Missing issue type", func() {
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
})

func GetStringAddress(str string) *string {
	return &str
}
