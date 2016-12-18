package go_rps_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGoRps(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GoRps Suite")
}
