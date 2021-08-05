package url

import (
	"net/url"
	"path"
)

type UserMultipassEndpointBuilder struct{}

func (b *UserMultipassEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/user", params["user"].(string), "/multipass") + "?" + query.Encode()
}

func (b *UserMultipassEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/user", params["user"].(string), "/multipass", params["multipass"].(string)) + "?" + query.Encode()
}

func (b *UserMultipassEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/user", params["user"].(string), "/multipass") + "/?" + query.Encode()
}

func (b *UserMultipassEndpointBuilder) Privileged(params Params, query url.Values) string {
	panic("this path is not allowed")
}
