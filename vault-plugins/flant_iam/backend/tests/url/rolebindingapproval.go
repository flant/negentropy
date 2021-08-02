package url

import (
	"net/url"
	"path"
)

type RoleBindingApprovalEndpointBuilder struct{}

func (b *RoleBindingApprovalEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant_uuid"].(string), "role_binding", params["role_binding_uuid"].(string), "approval", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *RoleBindingApprovalEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant_uuid"].(string), "role_binding", params["role_binding_uuid"].(string), "approval", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *RoleBindingApprovalEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant_uuid"].(string), "role_binding", params["role_binding_uuid"].(string), "approval") + "/?" + query.Encode()
}

func (b *RoleBindingApprovalEndpointBuilder) Privileged(_ Params, _ url.Values) string {
	panic("this path is not allowed")
}
