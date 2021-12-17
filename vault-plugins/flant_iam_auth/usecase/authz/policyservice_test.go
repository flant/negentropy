package authz

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Rule(t *testing.T) {
	r := &Rule{
		Path:   "/tenant/199908",
		Create: true,
		Update: true,
		Read:   true,
		Delete: true,
		List:   true,
	}

	s := r.String()

	require.Equal(t, "path \"/tenant/199908\" {\n   capabilities = [\"create\", \"update\", \"read\", \"delete\", \"list\"]\n}", s)
}
