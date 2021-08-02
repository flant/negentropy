package rolebindingapproval

import (
	"net/url"
	"path"

	"github.com/flant/negentropy/e2e/tests/lib/tools"
)

type EndpointBuilder struct{}

func (b *EndpointBuilder) OneCreate(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant_uuid"].(string), "role_binding", params["role_binding_uuid"].(string), "approval", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *EndpointBuilder) One(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant_uuid"].(string), "role_binding", params["role_binding_uuid"].(string), "approval", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *EndpointBuilder) Collection(params tools.Params, query url.Values) string {
	return path.Join("/tenant", params["tenant_uuid"].(string), "role_binding", params["role_binding_uuid"].(string), "approval") + "/?" + query.Encode()
}

func (b *EndpointBuilder) Privileged(_ tools.Params, _ url.Values) string {
	return "" // Not implemented
}
