package tools

import (
	"fmt"
	"go/token"
	"go/types"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func ExpectExactStatus(status int) func(response *http.Response) {
	return func(resp *http.Response) {
		Expect(resp.StatusCode).To(Equal(status))
	}
}

func ExpectStatus(condition string) func(response *http.Response) {
	return func(resp *http.Response) {
		formula := fmt.Sprintf(condition, resp.StatusCode)
		By("Status code check "+formula, func() {
			fs := token.NewFileSet()

			tv, err := types.Eval(fs, nil, token.NoPos, formula)
			Expect(err).ToNot(HaveOccurred())

			Expect(tv.Value.String()).To(Equal("true"))
		})

	}
}
