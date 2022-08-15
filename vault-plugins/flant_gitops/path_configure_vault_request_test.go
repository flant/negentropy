package flant_gitops

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PathConfigureVaultRequestCallbacksSuite struct {
	suite.Suite
	ctx     context.Context
	backend *backend
	request *logical.Request
}

var fullValidVaultRequest = &vaultRequest{
	Name:   "request1",
	Path:   "/path",
	Method: "POST",
	Options: map[string]interface{}{
		"str":  "str",
		"list": []interface{}{"str"},
		"map":  map[string]interface{}{"str": "str"},
	},
	WrapTTL: time.Duration(60) * time.Second,
}

func (s *PathConfigureVaultRequestCallbacksSuite) SetupTest() {
	b := &backend{}
	b.Backend = &framework.Backend{
		Paths: configureVaultRequestPaths(b),
	}

	ctx := context.Background()
	storage := &logical.InmemStorage{}
	config := logical.TestBackendConfig()
	config.StorageView = storage

	err := b.Setup(ctx, config)
	assert.Nil(s.T(), err)

	request := &logical.Request{
		Path:    "configure/vault_request/request1",
		Storage: storage,
		Data:    map[string]interface{}{},
	}

	s.ctx = ctx
	s.backend = b
	s.request = request
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_CreateOrUpdate_NoRequestPath() {
	assert := assert.New(s.T())

	s.request.Operation = logical.CreateOperation
	s.request.Data = vaultRequestStructToMap(fullValidVaultRequest)
	s.request.Data[fieldNameVaultRequestPath] = ""

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	resp.Warnings = nil // delete warnings
	assert.Equal(logical.ErrorResponse(`%q field value must begin with "/", got: `, fieldNameVaultRequestPath), resp)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_CreateOrUpdate_InvalidRequestPath() {
	assert := assert.New(s.T())

	s.request.Operation = logical.CreateOperation
	s.request.Data = vaultRequestStructToMap(fullValidVaultRequest)
	s.request.Data[fieldNameVaultRequestPath] = "123"

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	resp.Warnings = nil // delete warnings
	assert.Equal(logical.ErrorResponse(`%q field value must begin with "/", got: 123`, fieldNameVaultRequestPath), resp)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_CreateOrUpdate_InvalidMethod() {
	assert := assert.New(s.T())

	s.request.Operation = logical.CreateOperation
	s.request.Data = vaultRequestStructToMap(fullValidVaultRequest)
	s.request.Data[fieldNameVaultRequestMethod] = "123"

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	resp.Warnings = nil // delete warnings
	assert.Equal(logical.ErrorResponse("%q field value must be one of GET, POST, LIST, PUT or DELETE, got: 123", fieldNameVaultRequestMethod), resp)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_CreateOrUpdate_WrapTTLTooSmall() {
	assert := assert.New(s.T())

	s.request.Operation = logical.CreateOperation
	s.request.Data = vaultRequestStructToMap(fullValidVaultRequest)
	s.request.Data[fieldNameVaultRequestWrapTTL] = "1"

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	resp.Warnings = nil // delete warnings
	assert.Equal(logical.ErrorResponse("%q field value must be no less than %ds, got: 1s", fieldNameVaultRequestWrapTTL, vaultRequestWrapTTLMinSec), resp)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_CreateOrUpdate_FullValidConfig() {
	assert := assert.New(s.T())

	s.request.Operation = logical.CreateOperation
	s.request.Data = vaultRequestStructToMap(fullValidVaultRequest)

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)

	vaultReq, err := getVaultRequest(s.ctx, s.request.Storage, fullValidVaultRequest.Name)
	assert.Nil(err)
	assert.Equal(fullValidVaultRequest, vaultReq)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_CreateOrUpdate_MultipleFullValidConfigs() {
	assert := assert.New(s.T())

	s.request.Operation = logical.CreateOperation
	s.request.Data = vaultRequestStructToMap(fullValidVaultRequest)

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)

	fullValidVaultRequest2 := *fullValidVaultRequest
	fullValidVaultRequest2.Name = "request2"
	request2 := &logical.Request{
		Path:      "configure/vault_request/request2",
		Storage:   s.request.Storage,
		Operation: logical.CreateOperation,
		Data:      vaultRequestStructToMap(&fullValidVaultRequest2),
	}

	resp, err = s.backend.HandleRequest(s.ctx, request2)
	assert.Nil(err)
	assert.Nil(resp)

	vaultReq, err := getVaultRequest(s.ctx, s.request.Storage, fullValidVaultRequest.Name)
	assert.Nil(err)
	assert.Equal(fullValidVaultRequest, vaultReq)

	vaultReq2, err := getVaultRequest(s.ctx, s.request.Storage, fullValidVaultRequest2.Name)
	assert.Nil(err)
	assert.Equal(&fullValidVaultRequest2, vaultReq2)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_Read_NoConfig() {
	assert := assert.New(s.T())

	s.request.Operation = logical.ReadOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_Read_HasConfig() {
	assert := assert.New(s.T())

	err := putVaultRequest(s.ctx, s.request.Storage, *fullValidVaultRequest)
	assert.Nil(err)

	s.request.Operation = logical.ReadOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Equal(&logical.Response{Data: vaultRequestStructToMap(fullValidVaultRequest)}, resp)

	vaultReq, err := getVaultRequest(s.ctx, s.request.Storage, fullValidVaultRequest.Name)
	assert.Nil(err)
	assert.Equal(fullValidVaultRequest, vaultReq)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_List_NoConfig() {
	assert := assert.New(s.T())

	s.request.Operation = logical.ListOperation
	s.request.Path = "configure/vault_request"

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_List_HasConfig() {
	assert := assert.New(s.T())

	err := putVaultRequest(s.ctx, s.request.Storage, *fullValidVaultRequest)
	assert.Nil(err)

	s.request.Operation = logical.ListOperation
	s.request.Path = "configure/vault_request"

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Equal(
		logical.ListResponseWithInfo(
			[]string{fullValidVaultRequest.Name},
			map[string]interface{}{fullValidVaultRequest.Name: vaultRequestStructToMap(fullValidVaultRequest)},
		),
		resp,
	)

	vaultReq, err := getVaultRequest(s.ctx, s.request.Storage, fullValidVaultRequest.Name)
	assert.Nil(err)
	assert.Equal(fullValidVaultRequest, vaultReq)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_Delete_NoConfig() {
	assert := assert.New(s.T())

	s.request.Operation = logical.DeleteOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)
}

func (s *PathConfigureVaultRequestCallbacksSuite) Test_Delete_HasConfig() {
	assert := assert.New(s.T())

	err := putVaultRequest(s.ctx, s.request.Storage, *fullValidVaultRequest)
	assert.Nil(err)

	s.request.Operation = logical.DeleteOperation

	resp, err := s.backend.HandleRequest(s.ctx, s.request)
	assert.Nil(err)
	assert.Nil(resp)

	vaultReq, err := getVaultRequest(s.ctx, s.request.Storage, fullValidVaultRequest.Name)
	assert.Nil(err)
	assert.Nil(vaultReq)
}

func TestPathConfigureVaultRequest(t *testing.T) {
	suite.Run(t, new(PathConfigureVaultRequestCallbacksSuite))
}
