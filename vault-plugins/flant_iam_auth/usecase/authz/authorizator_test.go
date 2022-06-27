package authz

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func Test_globalScope(t *testing.T) {
	scope, err := checkAndEvaluateScope(&model.Role{
		Scope:             model.RoleScopeTenant,
		TenantIsOptional:  true,
		ProjectIsOptional: false,
	}, "", "")

	require.NoError(t, err)
	require.Equal(t, globalScope, scope)
}
