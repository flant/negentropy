package url

import (
	"net/url"
	"path"
)

type FeatureFlagEndpointBuilder struct{}

func (b *FeatureFlagEndpointBuilder) OneCreate(_ Params, query url.Values) string {
	return "feature_flag"
}

func (b *FeatureFlagEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("feature_flag", params["name"].(string)) + "?" + query.Encode()
}

func (b *FeatureFlagEndpointBuilder) Collection(_ Params, query url.Values) string {
	return path.Join("feature_flag") + "/?" + query.Encode()
}

func (b *FeatureFlagEndpointBuilder) Privileged(_ Params, _ url.Values) string {
	panic("this path is not allowed")
}
