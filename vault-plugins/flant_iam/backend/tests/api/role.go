package api

import (
	"net/url"
	"path"

	"github.com/hashicorp/vault/sdk/logical"
)

type RoleAPI struct {
	b logical.Backend
}

type RoleEndpointBuilder struct{}

func (b *RoleEndpointBuilder) OneCreate(Params, url.Values) string {
	return "role"
}

func (b *RoleEndpointBuilder) One(params Params, _ url.Values) string {
	return path.Join("role", params["name"].(string))
}

func (b *RoleEndpointBuilder) Collection(_ Params, _ url.Values) string {
	return "role/"
}

func (b *RoleEndpointBuilder) Privileged(_ Params, _ url.Values) string {
	panic("this path is not allowed")
}
