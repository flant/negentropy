package url

import (
	"net/url"
	"path"
)

type RoleBindingApprovalEndpointBuilder struct{}

func (b *RoleBindingApprovalEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "role_binding", params["role_binding"].(string), "approval", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *RoleBindingApprovalEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "role_binding", params["role_binding"].(string), "approval", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *RoleBindingApprovalEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "role_binding", params["role_binding"].(string), "approval") + "/?" + query.Encode()
}

func (b *RoleBindingApprovalEndpointBuilder) Privileged(_ Params, _ url.Values) string {
	panic("this path is not allowed")
}
