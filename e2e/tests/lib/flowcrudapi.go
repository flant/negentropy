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
)

func NewTeamAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.TeamEndpointBuilder{}}
}

func NewTeammateAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.TeammateEndpointBuilder{}}
}

func NewClientAPI(client *http.Client) TestAPI {
	return &BuilderBasedAPI{client: client, url: &url2.ClientEndpointBuilder{}}
}
