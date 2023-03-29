package generate

import (
	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"
)

var _ = Describe("Generate Grafana Test", func() {
	Context("normalize data test", func() {
		When("a path has only one set of brackets", func() {
			It("should replace the brackets and contents with hyphen", func() {
				normalizedPath := normalizePath("/api/awesome_mgmt/v1/noop/{id}")
				Expect(normalizedPath).To(Equal("/api/awesome_mgmt/v1/noop/-"))
			})
		})

		When("a path has two sets of brackets", func() {
			It("should replace the individual brackets and contents with a hyphen", func() {
				normalizedPath := normalizePath("/api/awesome_mgmt/v1/noop/{id}/boop/{another_id}")
				Expect(normalizedPath).To(Equal("/api/awesome_mgmt/v1/noop/-/boop/-"))
			})
		})

		When("a path has multiple sets of brackets", func() {
			It("should replace the individual brackets and contents with a hyphen", func() {
				normalizedPath := normalizePath("/api/awesome_mgmt/v1/noop/{id}/boop/{another_id}/moop/{yet_another_id}")
				Expect(normalizedPath).To(Equal("/api/awesome_mgmt/v1/noop/-/boop/-/moop/-"))
			})
		})
	})
})
