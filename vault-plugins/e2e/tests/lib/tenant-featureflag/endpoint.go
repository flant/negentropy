package tenant_featureflag

import (
	"net/url"
	"path"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
)

type EndpointBuilder struct{}

func (b *EndpointBuilder) One(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant_uuid"].(string), "feature_flag", params["feature_flag_name"].(string)) + "?" + query.Encode()
}

func (b *EndpointBuilder) Collection(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant_uuid"].(string), "feature_flag", params["feature_flag_name"].(string)) + "?" + query.Encode()
}

func (b *EndpointBuilder) Privileged(_ tools.Params, query url.Values) string {
	return ""
}
