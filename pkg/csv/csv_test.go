package csv

import (
	"testing"

	. "github.com/onsi/ginkgo/v2" // nolint
	. "github.com/onsi/gomega"    // nolint
)

func TestCsv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Csv")
}
