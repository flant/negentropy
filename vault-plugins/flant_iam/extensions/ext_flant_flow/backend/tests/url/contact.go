package url

import (
	"net/url"
	"path"
)

type ContactEndpointBuilder struct{}

func (b *ContactEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("client", params["client"].(string), "/contact") + "?" + query.Encode()
}

func (b *ContactEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("client", params["client"].(string), "/contact", params["contact"].(string)) + "?" + query.Encode()
}

func (b *ContactEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("client", params["client"].(string), "/contact") + "/?" + query.Encode()
}

func (b *ContactEndpointBuilder) Privileged(params Params, query url.Values) string {
	return path.Join("client", params["client"].(string), "/contact", "privileged") + "?" + query.Encode()
}
