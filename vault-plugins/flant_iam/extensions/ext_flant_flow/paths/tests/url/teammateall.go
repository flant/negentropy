package url

import (
	"net/url"
	"path"
)

type TeammateListAllEndpointBuilder struct{}

func (b *TeammateListAllEndpointBuilder) OneCreate(_ Params, _ url.Values) string {
	panic("this path is not allowed")
}

func (b *TeammateListAllEndpointBuilder) One(_ Params, _ url.Values) string {
	panic("this path is not allowed")
}

func (b *TeammateListAllEndpointBuilder) Collection(_ Params, query url.Values) string {
	return path.Join("teammate") + "/?" + query.Encode()
}

func (b *TeammateListAllEndpointBuilder) Privileged(_ Params, _ url.Values) string {
	panic("this path is not allowed")
}
