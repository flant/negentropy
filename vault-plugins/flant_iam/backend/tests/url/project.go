package url

import (
	"net/url"
	"path"
)

type ProjectEndpointBuilder struct{}

func (b *ProjectEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/project") + "?" + query.Encode()
}

func (b *ProjectEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/project", params["project"].(string)) + "?" + query.Encode()
}

func (b *ProjectEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/project") + "/?" + query.Encode()
}

func (b *ProjectEndpointBuilder) Privileged(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "/project", "privileged") + "?" + query.Encode()
}
