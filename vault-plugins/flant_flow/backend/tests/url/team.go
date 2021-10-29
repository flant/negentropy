package url

import (
	"net/url"
	"path"
)

type TeamEndpointBuilder struct{}

func (b *TeamEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("team") + "?" + query.Encode()
}

func (b *TeamEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("team", params["team"].(string)) + "?" + query.Encode()
}

func (b *TeamEndpointBuilder) Collection(_ Params, query url.Values) string {
	return path.Join("team") + "/?" + query.Encode()
}

func (b *TeamEndpointBuilder) Privileged(_ Params, query url.Values) string {
	return path.Join("team", "privileged") + "?" + query.Encode()
}
