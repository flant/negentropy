package url

import (
	"net/url"
	"path"

	api "github.com/flant/negentropy/vault-plugins/shared/tests"
)

type PolicyEndpointBuilder struct{}

func (b *PolicyEndpointBuilder) OneCreate(params api.Params, query url.Values) string {
	return path.Join("login_policy") + "?" + query.Encode()
}

func (b *PolicyEndpointBuilder) One(params api.Params, query url.Values) string {
	return path.Join("login_policy", params["policy"].(string)) + "?" + query.Encode()
}

func (b *PolicyEndpointBuilder) Collection(_ api.Params, query url.Values) string {
	return path.Join("login_policy") + "/?" + query.Encode()
}

func (b *PolicyEndpointBuilder) Privileged(_ api.Params, query url.Values) string {
	panic("this path is nor allowed")
}
