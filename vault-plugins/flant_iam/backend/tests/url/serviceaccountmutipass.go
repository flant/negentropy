package url

import (
	"net/url"
	"path"
)

type ServiceAccountMultipassEndpointBuilder struct{}

func (b *ServiceAccountMultipassEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/service_account", params["service_account"].(string), "/multipass") + "?" + query.Encode()
}

func (b *ServiceAccountMultipassEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/service_account", params["service_account"].(string), "/multipass", params["multipass"].(string)) + "?" + query.Encode()
}

func (b *ServiceAccountMultipassEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/service_account", params["service_account"].(string), "/multipass") + "/?" + query.Encode()
}

func (b *ServiceAccountMultipassEndpointBuilder) Privileged(params Params, query url.Values) string {
	panic("this path is not allowed")
}
