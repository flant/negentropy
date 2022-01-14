package url

import (
	"net/url"
	"path"
)

type ServiceAccountEndpointBuilder struct{}

func (b *ServiceAccountEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/service_account") + "?" + query.Encode()
}

func (b *ServiceAccountEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/service_account", params["service_account"].(string)) + "?" + query.Encode()
}

func (b *ServiceAccountEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/service_account") + "/?" + query.Encode()
}

func (b *ServiceAccountEndpointBuilder) Privileged(_ Params, query url.Values) string {
	panic("this path is not allowed")
}
