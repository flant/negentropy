package backentutils

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

func Test_MapErrorToHTTPStatusCode(t *testing.T) {
	type testCase struct {
		err               error
		expecedStatusCode int
	}
	testcases := []testCase{
		{consts.ErrNotConfigured, http.StatusPreconditionRequired},
		{fmt.Errorf("%w:some extra text", consts.ErrNotConfigured), http.StatusPreconditionRequired},
		{fmt.Errorf("some extra text:%w", consts.ErrNotConfigured), http.StatusPreconditionRequired},
	}
	for _, tc := range testcases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			assert.Equal(t, tc.expecedStatusCode, MapErrorToHTTPStatusCode(tc.err))
		})
	}
}
