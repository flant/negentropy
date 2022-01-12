package url

import (
	"net/url"
	"path"
)

type ServiceAccountPasswordEndpointBuilder struct{}

func (b *ServiceAccountPasswordEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/service_account", params["service_account"].(string), "/password") + "?" + query.Encode()
}

func (b *ServiceAccountPasswordEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/service_account", params["service_account"].(string), "/password", params["password"].(string)) + "?" + query.Encode()
}

func (b *ServiceAccountPasswordEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/service_account", params["service_account"].(string), "/password") + "/?" + query.Encode()
}

func (b *ServiceAccountPasswordEndpointBuilder) Privileged(params Params, query url.Values) string {
	panic("this path is not allowed")
}
