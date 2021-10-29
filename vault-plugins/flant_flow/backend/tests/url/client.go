package url

import (
	"net/url"
	"path"
)

type ClientEndpointBuilder struct{}

func (b *ClientEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("client") + "?" + query.Encode()
}

func (b *ClientEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("client", params["client"].(string)) + "?" + query.Encode()
}

func (b *ClientEndpointBuilder) Collection(_ Params, query url.Values) string {
	return path.Join("client") + "/?" + query.Encode()
}

func (b *ClientEndpointBuilder) Privileged(_ Params, query url.Values) string {
	return path.Join("client", "privileged") + "?" + query.Encode()
}
