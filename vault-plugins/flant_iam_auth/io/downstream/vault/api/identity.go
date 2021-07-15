package api

import (
	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
)

type IdentityAPI struct {
	clientApi     *api.Client
	backoffGetter func() backoff.BackOff
	logger        hclog.Logger
}

func NewIdentityAPI(clientApi *api.Client, logger hclog.Logger) *IdentityAPI {
	return &IdentityAPI{
		clientApi: clientApi,
		logger:    logger,
	}
}

func NewIdentityAPIWithBackOff(clientApi *api.Client, backoffGetter func() backoff.BackOff) *IdentityAPI {
	return &IdentityAPI{
		clientApi:     clientApi,
		backoffGetter: backoffGetter,
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
