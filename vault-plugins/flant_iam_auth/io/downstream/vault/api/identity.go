package api

import (
	"github.com/cenkalti/backoff"
	"github.com/hashicorp/vault/api"
)

type IdentityAPI struct {
	clientApi     *api.Client
	backoffGetter func() backoff.BackOff
}

func NewIdentityAPI(clientApi *api.Client) *IdentityAPI {
	return &IdentityAPI{
		clientApi: clientApi,
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
