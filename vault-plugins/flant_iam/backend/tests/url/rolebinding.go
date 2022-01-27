package url

import (
	"net/url"
	"path"
)

type RoleBindingEndpointBuilder struct{}

func (b *RoleBindingEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "role_binding") + "?" + query.Encode()
}

func (b *RoleBindingEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "role_binding", params["role_binding"].(string)) + "?" + query.Encode()
}

func (b *RoleBindingEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "role_binding") + "/?" + query.Encode()
}

func (b *RoleBindingEndpointBuilder) Privileged(_ Params, _ url.Values) string {
	panic("this path is not allowed")
}
