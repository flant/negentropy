package url

import (
	"net/url"
	"path"
)

type TenantFeatureFlagEndpointBuilder struct{}

func (b *TenantFeatureFlagEndpointBuilder) OneCreate(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "feature_flag", params["feature_flag_name"].(string)) + "?" + query.Encode()
}

func (b *TenantFeatureFlagEndpointBuilder) One(params Params, query url.Values) string {
	return path.Join("tenant", params["tenant"].(string), "feature_flag", params["feature_flag_name"].(string)) + "?" + query.Encode()
}

func (b *TenantFeatureFlagEndpointBuilder) Collection(params Params, query url.Values) string {
	panic("this path is not allowed")
}

func (b *TenantFeatureFlagEndpointBuilder) Privileged(_ Params, query url.Values) string {
	panic("this path is not allowed")
}
