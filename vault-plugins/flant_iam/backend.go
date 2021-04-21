package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/pborman/uuid"
)

const (
	tenantUUID = "id"
)

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := &framework.Backend{
		Help:        strings.TrimSpace(commonHelp),
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

		/*{
			Pattern: "tenant/new",
			Fields: map[string]*framework.FieldSchema{
				"name": {Type: framework.TypeString, Description: "Tenant name"},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.create,
					Summary:  "Create a tenant.",
				},
			},
		},*/
		{
			Pattern: "tenant" + framework.OptionalParamRegex(tenantUUID),
			Fields: map[string]*framework.FieldSchema{
				tenantUUID: {Type: framework.TypeString, Description: "ID of a tenant"},
				"name":     {Type: framework.TypeString, Description: "Tenant name"},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
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
			//ExistenceCheck: b.handleExistenceCheck,
		},
		//{
		//	Pattern: "tenant/?",
		//	Operations: map[logical.Operation]framework.OperationHandler{
		//		logical.ListOperation: &framework.PathOperation{
		//			Callback: b.handleList,
		//			Summary:  "Lists all tenants IDs.",
		//		},
		//	},
		//},
	}
}

type backend struct{}

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

	id := data.Get(tenantUUID).(string)

	// Decode the data
	var rawData map[string]interface{}
	fetchedData, err := req.Storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if fetchedData == nil {
		errResp := logical.ErrorResponse("No value at %v%v", req.MountPoint, id)
		resp, _ := logical.RespondWithStatusCode(errResp, req, 404)
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

	path := data.Get(tenantUUID).(string)

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

	successStatus := http.StatusOK
	rawID, ok := data.GetOk(tenantUUID)
	if !ok {
		// the creation here
		rawID = uuid.New()
		successStatus = http.StatusCreated
	}
	id := rawID.(string)

	name, ok := data.GetOk("name")
	if !ok {
		return nil, &logical.StatusBadRequest{Err: "tenant name must not be empty"}
	}
	if len(name.(string)) == 0 {
		return nil, &logical.StatusBadRequest{Err: "tenant name must not be empty"}
	}

	// JSON encode the data
	buf, err := json.Marshal(req.Data)
	if err != nil {
		return nil, errwrap.Wrapf("json encoding failed: {{err}}", err)
	}

	entry := &logical.StorageEntry{
		Key:   id,
		Value: buf,
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"id": id,
		},
	}
	return logical.RespondWithStatusCode(resp, req, successStatus)
}

func (b *backend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if req.ClientToken == "" {
		return nil, fmt.Errorf("client token empty")
	}
	path := data.Get(tenantUUID).(string)
	err := req.Storage.Delete(ctx, path)

	return nil, err
}

const commonHelp = `
IAM API here
`
