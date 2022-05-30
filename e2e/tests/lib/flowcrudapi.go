package lib

import (
	"net/http"

	url2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/url"
)

var (
	_ TestAPI = (*BuilderBasedAPI)(nil)

	_ URLBuilder = (*url2.TeamEndpointBuilder)(nil)
	_ URLBuilder = (*url2.TeammateEndpointBuilder)(nil)
	_ URLBuilder = (*url2.ClientEndpointBuilder)(nil)
	_ URLBuilder = (*url2.ProjectEndpointBuilder)(nil)
	_ URLBuilder = (*url2.ContactEndpointBuilder)(nil)
	_ URLBuilder = (*url2.TeammateListAllEndpointBuilder)(nil)
)

func NewFlowTeamAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.TeamEndpointBuilder{}}
}

func NewFlowTeammateAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.TeammateEndpointBuilder{}}
}

func NewFlowClientAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ClientEndpointBuilder{}}
}

func NewFlowProjectAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ProjectEndpointBuilder{}}
}

func NewFlowContactAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ContactEndpointBuilder{}}
}

func NewFlowTeammateListAllAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.TeammateListAllEndpointBuilder{}}
}
