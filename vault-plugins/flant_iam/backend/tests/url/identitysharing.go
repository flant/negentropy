package url

import (
	"net/url"
	"path"
)

type IdentitySharingEndpointBuilder struct{}

func (b *IdentitySharingEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant_uuid"].(string), "identity_sharing") + "?" + query.Encode()
}

func (b *IdentitySharingEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant_uuid"].(string), "identity_sharing", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *IdentitySharingEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant_uuid"].(string), "identity_sharing") + "/?" + query.Encode()
}

func (b *IdentitySharingEndpointBuilder) Privileged(_ Params, query url.Values) string {
	panic("this path is not allowed")
}
