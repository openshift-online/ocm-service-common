package ocmlogger_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOcmlogger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCMLogger Suite")
}
