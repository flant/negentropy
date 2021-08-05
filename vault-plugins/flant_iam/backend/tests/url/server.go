package url

import (
	"net/url"
	"path"
)

type ServerEndpointBuilder struct{}

func (b *ServerEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/project", params["project"].(string),
		"register_server") + "?" + query.Encode()
}

func (b *ServerEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/project", params["project"].(string),
		"server", params["uuid"].(string)) + "?" + query.Encode()
}

func (b *ServerEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/project", params["project"].(string),
		"servers") + "/?" + query.Encode()
}

func (b *ServerEndpointBuilder) Privileged(params Params, query url.Values) string {
	panic("this path is not allowed")
}
