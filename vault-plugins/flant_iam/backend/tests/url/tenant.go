package url

import (
	"net/url"
	"path"
)

type TenantEndpointBuilder struct{}

func (b *TenantEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant") + "?" + query.Encode()
}

func (b *TenantEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string)) + "?" + query.Encode()
}

func (b *TenantEndpointBuilder) Collection(_ Params, query url.Values) string {
	return path.Join("tenant") + "/?" + query.Encode()
}

func (b *TenantEndpointBuilder) Privileged(_ Params, query url.Values) string {
	return path.Join("tenant", "privileged") + "?" + query.Encode()
}
