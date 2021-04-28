package flant_gitops

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type backend struct {
	*framework.Backend
}

var _ logical.Factory = Factory

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := newBackend()
	if err != nil {
		return nil, err
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend() (*backend, error) {
	b := &backend{}
	b.Backend = &framework.Backend{
		PeriodicFunc: b.periodic,
		BackendType:  logical.TypeLogical,
		Paths: framework.PathAppend(
			[]*framework.Path{
				pathConfigure(b),
			},
		),
	}

	return b, nil
}
