package git_repository

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PathConfigureCallbacksSuite struct {
	suite.Suite
	ctx     context.Context
	backend *framework.Backend
	request *logical.Request
}

var fullValidConfiguration = &Configuration{
	GitRepoUrl:    "https://github.com/werf/vault-plugin-secrets-trdl.git",
	GitBranch:     "master",
	GitPollPeriod: time.Duration(60) * time.Second,
	RequiredNumberOfVerifiedSignaturesOnCommit: 0,
	InitialLastSuccessfulCommit:                "",
}

func (s *PathConfigureCallbacksSuite) SetupTest() {
	b := &framework.Backend{}
	storage := &logical.InmemStorage{}
	config := logical.TestBackendConfig()
	config.StorageView = storage

	ctx := context.Background()
	err := b.Setup(ctx, config)
	assert.Nil(s.T(), err)

	b.Paths = ConfigurePaths(b)
	request := &logical.Request{
		Path:    "configure",
		Storage: storage,
		Data:    map[string]interface{}{},
	}

	s.ctx = ctx
	s.backend = b
	s.request = request
}

func (s *PathConfigureCallbacksSuite) Test_CreateOrUpdate_NoGitRepoURL() {
	assert := assert.New(s.T())

	s.request.Operation = logical.CreateOperation
	s.request.Data = configurationStructToMap(fullValidConfiguration)
	s.request.Data[FieldNameGitRepoUrl] = ""

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Equal(logical.ErrorResponse("%q field value should not be empty", FieldNameGitRepoUrl), resp)
}

func (s *PathConfigureCallbacksSuite) Test_CreateOrUpdate_FullValidConfig() {
	assert := assert.New(s.T())

	s.request.Operation = logical.CreateOperation
	s.request.Data = configurationStructToMap(fullValidConfiguration)

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)

	cfg, err := getConfiguration(s.ctx, s.request.Storage)
	assert.Nil(err)
	assert.Equal(fullValidConfiguration, cfg)
}

func (s *PathConfigureCallbacksSuite) Test_Read_NoConfig() {
	assert := assert.New(s.T())

	s.request.Operation = logical.ReadOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)
}

func (s *PathConfigureCallbacksSuite) Test_Read_HasConfig() {
	assert := assert.New(s.T())

	err := putConfiguration(s.ctx, s.request.Storage, *fullValidConfiguration)
	assert.Nil(err)

	s.request.Operation = logical.ReadOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Equal(&logical.Response{Data: configurationStructToMap(fullValidConfiguration)}, resp)

	cfg, err := getConfiguration(s.ctx, s.request.Storage)
	assert.Nil(err)
	assert.Equal(fullValidConfiguration, cfg)
}

func (s *PathConfigureCallbacksSuite) Test_Delete_NoConfig() {
	assert := assert.New(s.T())

	s.request.Operation = logical.DeleteOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)
}

func (s *PathConfigureCallbacksSuite) Test_Delete_HasConfig() {
	assert := assert.New(s.T())

	err := putConfiguration(s.ctx, s.request.Storage, *fullValidConfiguration)
	assert.Nil(err)

	s.request.Operation = logical.DeleteOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)

	cfg, err := getConfiguration(s.ctx, s.request.Storage)
	assert.Nil(err)
	assert.Nil(cfg)
}

func TestPathConfigure(t *testing.T) {
	suite.Run(t, new(PathConfigureCallbacksSuite))
}
