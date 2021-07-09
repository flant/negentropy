package featureflag

import (
	"net/url"
	"path"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
)

type EndpointBuilder struct{}

func (b *EndpointBuilder) One(params tools.Params, query url.Values) string {
	return path.Join("/feature_flag", params["name"].(string)) + "?" + query.Encode()
}

func (b *EndpointBuilder) Collection(_ tools.Params, query url.Values) string {
	return path.Join("/feature_flag") + "?" + query.Encode()
}

func (b *EndpointBuilder) Privileged(_ tools.Params, _ url.Values) string {
	return "" // Not implemented
}
