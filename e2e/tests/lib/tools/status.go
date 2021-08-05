package tools

import (
	"fmt"
	"go/token"
	"go/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func ExpectExactStatus(expectedStatus int) func(int) {
	return func(statusCode int) {
		Expect(statusCode).To(Equal(expectedStatus))
	}
}

func ExpectStatus(condition string) func(int) {
	return func(statusCode int) {
		formula := fmt.Sprintf(condition, statusCode)
		By("Status code check "+formula, func() {
			fs := token.NewFileSet()

			tv, err := types.Eval(fs, nil, token.NoPos, formula)
			Expect(err).ToNot(HaveOccurred())

			Expect(tv.Value.String()).To(Equal("true"))
		})
	}
}
