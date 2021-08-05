package url

import (
	"net/url"
	"path"
)

type GroupEndpointBuilder struct{}

func (b *GroupEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/group") + "?" + query.Encode()
}

func (b *GroupEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/group", params["group"].(string)) + "?" + query.Encode()
}

func (b *GroupEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/group") + "/?" + query.Encode()
}

func (b *GroupEndpointBuilder) Privileged(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/group", "privileged") + "?" + query.Encode()
}
