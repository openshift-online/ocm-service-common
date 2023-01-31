package jira

import (
	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"
)

// set const here, to ensure if `ProductsCustomField` changes the test will fail
const productsCustomField = "customfield_12319040"

var _ = Describe("Unknowns Test", func() {
	When("Unknown `Products` custom field is passed", func() {
		It("Should return the valid custom field", func() {
			knownField := getUnknownCustomField("Products")
			Expect(knownField).To(Equal(productsCustomField),
				"if this fails the products custom field has changed, check that the change is valid")
		})
	})
})


