package entity_and_alias

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	//t.Skip("Not configured in ci")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault entities")
}
