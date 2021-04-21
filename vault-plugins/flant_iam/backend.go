package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	tenantIdKey = "tid"
)

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := &framework.Backend{
		Help:        strings.TrimSpace(mockHelp),
		BackendType: logical.TypeLogical,
		Paths: framework.PathAppend(
			tenantPaths(),
		),
	}

	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func tenantPaths() []*framework.Path {
	b := backend{}
	return []*framework.Path{
		{
			Pattern: "tenant" + framework.OptionalParamRegex(tenantIdKey),
			Fields: map[string]*framework.FieldSchema{
				"path": {
					Type:        framework.TypeString,
					Description: "Specifies the path of the secret.",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList,
					Summary:  "Lists all tenants IDs.",
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleWrite,
					Summary:  "Create a tenant.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead,
					Summary:  "Retrieve the tenant by ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleWrite,
					Summary:  "Update the tenant by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete,
					Summary:  "Deletes the tenant by ID.",
				},
			},
			ExistenceCheck: b.handleExistenceCheck,
		},
	}
}

type backend struct {
}

func (b *backend) handleExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return false, errwrap.Wrapf("existence check failed: {{err}}", err)
	}

	return out != nil, nil
}

func (b *backend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("client token empty")
	}

	path := data.Get(tenantIdKey).(string)

	// Decode the data
	var rawData map[string]interface{}
	fetchedData, err := req.Storage.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	if fetchedData == nil {
		resp := logical.ErrorResponse("No value at %v%v", req.MountPoint, path)
		return resp, nil
	}

	if err := jsonutil.DecodeJSON(fetchedData.Value, &rawData); err != nil {
		return nil, errwrap.Wrapf("json decoding failed: {{err}}", err)
	}

	// Generate the response
	resp := &logical.Response{
		Data: rawData,
	}

	return resp, nil
}

func (b *backend) handleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("client token empty")
	}

	path := data.Get(tenantIdKey).(string)

	// Decode the data
	fetchedData, err := req.Storage.List(ctx, path)
	if err != nil {
		return nil, err
	}
	if fetchedData == nil {
		resp := logical.ErrorResponse("No value in the list %v%v", req.MountPoint, path)
		return resp, nil
	}

	// Generate the response
	resp := &logical.Response{
		Data: map[string]interface{}{
			"keys": fetchedData,
			"peys": fetchedData,
		},
	}

	return resp, nil
}

func (b *backend) handleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("client token empty")
	}

	// Check to make sure that kv pairs provided
	if len(req.Data) == 0 {
		return nil, fmt.Errorf("data must be provided to store in secret")
	}

	// JSON encode the data
	buf, err := json.Marshal(req.Data)
	if err != nil {
		return nil, errwrap.Wrapf("json encoding failed: {{err}}", err)
	}

	path := data.Get(tenantIdKey).(string)
	key := path
	entry := &logical.StorageEntry{
		Key:   key,
		Value: buf,
	}
	err = req.Storage.Put(ctx, entry)

	return nil, err
}

func (b *backend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("client token empty")
	}
	path := data.Get(tenantIdKey).(string)
	err := req.Storage.Delete(ctx, path)

	return nil, err
}

const mockHelp = `
The Mock backend is a dummy secrets backend that stores kv pairs in a map.
`
