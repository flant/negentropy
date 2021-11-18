package api

import (
	"github.com/hashicorp/vault/sdk/logical"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	url2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/url"
)

func NewClientAPI(b *logical.Backend) testapi.TestAPI {
	return &testapi.BackendBasedAPI{Backend: b, Url: &url2.ClientEndpointBuilder{}}
}

func NewTeamAPI(b *logical.Backend) testapi.TestAPI {
	return &testapi.BackendBasedAPI{Backend: b, Url: &url2.TeamEndpointBuilder{}}
}

func NewTeammateAPI(b *logical.Backend) testapi.TestAPI {
	return &testapi.BackendBasedAPI{Backend: b, Url: &url2.TeammateEndpointBuilder{}}
}

func NewProjectAPI(b *logical.Backend) testapi.TestAPI {
	return &testapi.BackendBasedAPI{Backend: b, Url: &url2.ProjectEndpointBuilder{}}
}

func NewContactAPI(b *logical.Backend) testapi.TestAPI {
	return &testapi.BackendBasedAPI{Backend: b, Url: &url2.ContactEndpointBuilder{}}
}
