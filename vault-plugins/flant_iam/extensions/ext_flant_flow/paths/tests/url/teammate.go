package url

import (
	"net/url"
	"path"
)

type TeammateEndpointBuilder struct{}

func (b *TeammateEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("team", params["team"].(string), "/teammate") + "?" + query.Encode()
}

func (b *TeammateEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("team", params["team"].(string), "/teammate", params["teammate"].(string)) + "?" + query.Encode()
}

func (b *TeammateEndpointBuilder) Collection(params Params, query url.Values) string {
	return path.Join("team", params["team"].(string), "/teammate") + "/?" + query.Encode()
}

func (b *TeammateEndpointBuilder) Privileged(params Params, query url.Values) string {
	return path.Join("team", params["team"].(string), "/teammate", "privileged") + "?" + query.Encode()
}
