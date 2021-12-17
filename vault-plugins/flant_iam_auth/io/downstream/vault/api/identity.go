package api

import (
	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/shared/client"
)

type IdentityAPI struct {
	vaultClientProvider client.VaultClientController
	backoffGetter       func() backoff.BackOff
	logger              hclog.Logger
}

func NewIdentityAPI(vaultClientProvider client.VaultClientController, logger hclog.Logger) *IdentityAPI {
	return &IdentityAPI{
		vaultClientProvider: vaultClientProvider,
		logger:              logger,
	}
}

func NewIdentityAPIWithBackOff(vaultClientProvider client.VaultClientController, backoffGetter func() backoff.BackOff) *IdentityAPI {
	return &IdentityAPI{
		vaultClientProvider: vaultClientProvider,
		backoffGetter:       backoffGetter,
	}
}

func (a *IdentityAPI) EntityApi() *EntityAPI {
	return &EntityAPI{IdentityAPI: a}
}

func (a *IdentityAPI) AliasApi() *AliasAPI {
	return &AliasAPI{IdentityAPI: a}
}

func (a *IdentityAPI) callOp(op func() error) error {
	if a.backoffGetter != nil {
		return backoff.Retry(op, a.backoffGetter())
	}

	return op()
}
