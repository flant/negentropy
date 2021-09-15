package url

import (
	"net/url"
	"path"
)

type ProjectFeatureFlagEndpointBuilder struct{}

func (b *ProjectFeatureFlagEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "project", params["project"].(string),
		"feature_flag", params["feature_flag_name"].(string)) + "?" + query.Encode()
}

func (b *ProjectFeatureFlagEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "project", params["project"].(string),
		"feature_flag", params["feature_flag_name"].(string)) + "?" + query.Encode()
}

func (b *ProjectFeatureFlagEndpointBuilder) Collection(params Params, query url.Values) string {
	panic("this path is not allowed")
}

func (b *ProjectFeatureFlagEndpointBuilder) Privileged(_ Params, query url.Values) string {
	panic("this path is not allowed")
}
