package url

import (
	"net/url"
	"path"
)

type UserEndpointBuilder struct{}

func (b *UserEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/user") + "?" + query.Encode()
}

func (b *UserEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/user", params["user"].(string)) + "?" + query.Encode()
}

func (b *UserEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/user") + "/?" + query.Encode()
}

func (b *UserEndpointBuilder) Privileged(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/user", "privileged") + "?" + query.Encode()
}
