package api

import (
	"github.com/hashicorp/vault/sdk/logical"

	url2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/url"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

func NewClientAPI(b *logical.Backend, s *logical.Storage) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ClientEndpointBuilder{}, Storage: s}
}

func NewTeamAPI(b *logical.Backend, s *logical.Storage) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.TeamEndpointBuilder{}, Storage: s}
}

func NewTeammateAPI(b *logical.Backend, s *logical.Storage) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.TeammateEndpointBuilder{}, Storage: s}
}

func NewProjectAPI(b *logical.Backend, s *logical.Storage) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ProjectEndpointBuilder{}, Storage: s}
}

func NewContactAPI(b *logical.Backend, s *logical.Storage) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ContactEndpointBuilder{}, Storage: s}
}
