package url

import (
	"net/url"
	"path"
)

type UserEndpointBuilder struct{}

func (b *UserEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant") + "?" + query.Encode()
}

func (b *UserEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/user") + "?" + query.Encode()
}

func (b *UserEndpointBuilder) Collection(_ Params, query url.Values) string {
	return path.Join("tenant") + "/?" + query.Encode()
}

func (b *UserEndpointBuilder) Privileged(_ Params, query url.Values) string {
	return path.Join("tenant", "privileged", "/user") + "?" + query.Encode()
}
