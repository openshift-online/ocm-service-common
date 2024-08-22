package jira

import (
	// nolint

	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CustomField test", func() {
	It("Should marshall the Value of a Custom Field", func() {

		customField := NewCustomFieldType().Value("HyperShift Preview").Build()

		json, err := json.Marshal(customField)

		Expect(err).ToNot(HaveOccurred())
		Expect(json).To(ContainSubstring("\"value\":"))
	})
})
