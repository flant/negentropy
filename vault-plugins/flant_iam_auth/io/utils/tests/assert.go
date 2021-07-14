package tests

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
)

func AssertDeepEqual(t *testing.T, expected interface{}, val interface{}) {
	diff := deep.Equal(expected, val)
	require.Nil(t, diff)
}
