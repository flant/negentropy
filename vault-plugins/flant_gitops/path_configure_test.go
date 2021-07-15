package flant_gitops

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
)

type PathConfigureCallbacksSuite struct {
	suite.Suite
	ctx     context.Context
	backend *backend
	storage logical.Storage
	req     *logical.Request
}

var fullValidConfiguration = &configuration{
	GitRepoUrl:    "https://github.com/werf/vault-plugin-secrets-trdl.git",
	GitBranch:     "master",
	GitPollPeriod: time.Duration(1) * time.Second,
	RequiredNumberOfVerifiedSignaturesOnCommit: 0,
	InitialLastSuccessfulCommit:                "",
	DockerImage:                                "ubuntu:18.04@sha256:538529c9d229fb55f50e6746b119e899775205d62c0fc1b7e679b30d02ecb6e8",
	Commands:                                   []string{"echo Success"},
}

func (s *PathConfigureCallbacksSuite) SetupTest() {
	b := &backend{}
	b.Backend = &framework.Backend{
		Paths: configurePaths(b),
	}

	ctx := context.Background()
	storage := &logical.InmemStorage{}
	config := logical.TestBackendConfig()
	config.StorageView = storage

	err := b.Setup(ctx, config)
	assert.Nil(s.T(), err)

	req := &logical.Request{
		Path:    "configure",
		Storage: storage,
		Data:    map[string]interface{}{},
	}

	s.ctx = ctx
	s.backend = b
	s.storage = storage
	s.req = req
}

func (s *PathConfigureCallbacksSuite) Test_CreateOrUpdate_NoGitRepoURL() {
	assert := assert.New(s.T())

	s.req.Operation = logical.CreateOperation
	s.req.Data = configurationStructToMap(fullValidConfiguration)
	s.req.Data[fieldNameGitRepoUrl] = ""

	resp, err := s.backend.HandleRequest(s.ctx, s.req)
	assert.Nil(err)
	assert.Equal(logical.ErrorResponse("%q field value should not be empty", fieldNameGitRepoUrl), resp)
}

func (s *PathConfigureCallbacksSuite) Test_CreateOrUpdate_InvalidImageName() {
	assert := assert.New(s.T())

	s.req.Operation = logical.CreateOperation
	s.req.Data = configurationStructToMap(fullValidConfiguration)
	s.req.Data[fieldNameDockerImage] = "alpine"

	resp, err := s.backend.HandleRequest(s.ctx, s.req)
	assert.Nil(err)
	assert.Equal(logical.ErrorResponse("%q field is invalid: %s", fieldNameDockerImage, docker.ErrImageNameWithoutRequiredDigest), resp)
}

func (s *PathConfigureCallbacksSuite) Test_CreateOrUpdate_FullValidConfig() {
	assert := assert.New(s.T())

	s.req.Operation = logical.CreateOperation
	s.req.Data = configurationStructToMap(fullValidConfiguration)

	resp, err := s.backend.HandleRequest(s.ctx, s.req)
	assert.Nil(err)
	assert.Nil(resp)

	cfg, err := getConfiguration(s.ctx, s.storage)
	assert.Nil(err)
	assert.Equal(fullValidConfiguration, cfg)
}

func (s *PathConfigureCallbacksSuite) Test_Read_NoConfig() {
	assert := assert.New(s.T())

	s.req.Operation = logical.ReadOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.req)
	assert.Nil(err)
	assert.Nil(resp)
}

func (s *PathConfigureCallbacksSuite) Test_Read_HasConfig() {
	assert := assert.New(s.T())

	err := putConfiguration(s.ctx, s.storage, *fullValidConfiguration)
	assert.Nil(err)

	s.req.Operation = logical.ReadOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.req)
	assert.Nil(err)
	assert.Equal(&logical.Response{Data: configurationStructToMap(fullValidConfiguration)}, resp)

	cfg, err := getConfiguration(s.ctx, s.storage)
	assert.Nil(err)
	assert.Equal(fullValidConfiguration, cfg)
}

func (s *PathConfigureCallbacksSuite) Test_Delete_NoConfig() {
	assert := assert.New(s.T())

	s.req.Operation = logical.DeleteOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.req)
	assert.Nil(err)
	assert.Nil(resp)
}

func (s *PathConfigureCallbacksSuite) Test_Delete_HasConfig() {
	assert := assert.New(s.T())

	err := putConfiguration(s.ctx, s.storage, *fullValidConfiguration)
	assert.Nil(err)

	s.req.Operation = logical.DeleteOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.req)
	assert.Nil(err)
	assert.Nil(resp)

	cfg, err := getConfiguration(s.ctx, s.storage)
	assert.Nil(err)
	assert.Nil(cfg)
}

func TestGitCredentials(t *testing.T) {
	suite.Run(t, new(PathConfigureCallbacksSuite))
}
