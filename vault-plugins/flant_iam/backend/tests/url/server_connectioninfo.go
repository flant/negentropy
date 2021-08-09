package url

import (
	"net/url"
	"path"
)

type ConnectionInfoEndpointBuilder struct{}

func (b *ConnectionInfoEndpointBuilder) OneCreate(params Params, query url.Values) string {
	panic("this path is not allowed")
}

// the only operation for connection_info is update
func (b *ConnectionInfoEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string),
		"/project", params["project"].(string),
		"server", params["server"].(string), "/connection_info") + "?" + query.Encode()
}

func (b *ConnectionInfoEndpointBuilder) Collection(params Params, query url.Values) string {
	panic("this path is not allowed")
}

func (b *ConnectionInfoEndpointBuilder) Privileged(params Params, query url.Values) string {
	panic("this path is not allowed")
}
