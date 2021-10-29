package lib

import (
	"net/http"

	url2 "github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/url"
)

var (
	_ TestAPI = (*BuilderBasedAPI)(nil)

	_ URLBuilder = (*url2.TeamEndpointBuilder)(nil)
	_ URLBuilder = (*url2.TeammateEndpointBuilder)(nil)
	_ URLBuilder = (*url2.ClientEndpointBuilder)(nil)
	_ URLBuilder = (*url2.ProjectEndpointBuilder)(nil)
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

func NewFlowProjectClientAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ProjectEndpointBuilder{}}
}
