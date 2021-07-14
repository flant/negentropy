package user

import (
	"net/url"
	"path"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
)

type EndpointBuilder struct{}

func (b *EndpointBuilder) One(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant"].(string), "/user") + "?" + query.Encode()
}

func (b *EndpointBuilder) Collection(_ tools.Params, query url.Values) string {
	return path.Join("/tenant") + "?" + query.Encode()
}

func (b *EndpointBuilder) Privileged(_ tools.Params, query url.Values) string {
	return path.Join("/tenant", "privileged", "/user") + "?" + query.Encode()
}
