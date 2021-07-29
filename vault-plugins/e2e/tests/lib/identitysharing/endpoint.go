package identitysharing

import (
	"net/url"
	"path"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
)

type EndpointBuilder struct{}

func (b *EndpointBuilder) OneCreate(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant_uuid"].(string), "identity_sharing") + "?" + query.Encode()
}

func (b *EndpointBuilder) One(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant_uuid"].(string), "identity_sharing", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *EndpointBuilder) Collection(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant_uuid"].(string), "identity_sharing") + "/?" + query.Encode()
}

func (b *EndpointBuilder) Privileged(_ tools.Params, query url.Values) string {
	return ""
}
