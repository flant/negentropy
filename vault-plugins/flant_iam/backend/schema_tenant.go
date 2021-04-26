package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
)

type tenantBackend struct {
	logical.Backend
}

func tenantPaths(b logical.Backend) []*framework.Path {
	bb := &tenantBackend{b}
	return bb.paths()
}

func (b tenantBackend) paths() []*framework.Path {

	return []*framework.Path{
		{
			// using optional param in order to cover creation endpoint with empty id
			Pattern: "tenant/" + UUIDParam("id"),
			Fields: map[string]*framework.FieldSchema{
				"id": {
					Type:        framework.TypeString,
					Description: "ID of a tenant",
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true, // seems to work for doc, not validation
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				// POST, create or update
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleWrite,
					Summary:  "Update the tenant by ID.",
				},
				// GET
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead,
					Summary:  "Retrieve the tenant by ID.",
				},
				// DELETE
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete,
					Summary:  "Deletes the tenant by ID.",
				},
			},
		},
		{
			Pattern: "tenant/?$",
			Operations: map[logical.Operation]framework.OperationHandler{
				// GET
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList,
					Summary:  "Lists all tenants IDs.",
				},
			},
		},
	}
}

func getTenantKey(id, path string) (string, bool) {
	isCreating := id == ""
	if isCreating {
		key := path + "/" + genUUID()
		return key, isCreating
	}
	return path, isCreating
}

type tenant struct {
	identifier string
}

func (b *tenantBackend) handleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// creating or updating?
	id := data.Get("id").(string)
	key, isCreating := getTenantKey(id, req.Path)

	entry := &tenant{
		identifier: data.Get("identifier").(string),
	}

	// Validation

	// TODO: validation should depend on the storage
	//      validate field uniqueness
	//      validate resource_version

	//isUpdating := !isCreating
	//if isUpdating && err == ErrNotFound {
	//	// nothing to update
	//	errResp := logical.ErrorResponse("No value at %v%v", req.MountPoint, key)
	//	resp, _ := logical.RespondWithStatusCode(errResp, req, 404)
	//	return resp, nil
	//}
	//
	//b.Logger().Debug("writing", "key", key)
	//
	//err = b.schema.Validate(data)
	//if err != nil {
	//	return nil, &logical.StatusBadRequest{Err: err.Error()}
	//}

	//// Storing
	//input, err := b.schema.ParseData(data)
	//if err != nil {
	//	return nil, err
	//}

	// TODO resource version
	// if stored.Version() > input.Version() {
	//	return nil, &logical.StatusBadRequest{Err: "input resource version is older than stored one"}
	// }

	json, err := jsonutil.EncodeJSON(entry)
	if err != nil {
		return nil, err
	}

	req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   key,
		Value: json,
	})

	// Response

	resp := &logical.Response{
		Data: map[string]interface{}{
			"id": id,
		},
	}

	successStatus := http.StatusOK
	if isCreating {
		successStatus = http.StatusCreated
	}
	return logical.RespondWithStatusCode(resp, req, successStatus)
}

func (b *tenantBackend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	key := req.Path
	b.Logger().Debug("handleDelete", "key", key)

	// Ensure it exists
	found, err := req.Storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return errNotFoundResponse(req, key), nil
	}

	// Deletion

	// TODO: cascade deletion, e.g. deleteTenant()

	err = req.Storage.Delete(ctx, key)
	if err != nil {
		return nil, err
	}

	return nil, err
}

func (b *tenantBackend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("handleRead", "path", req.Path)
	key := req.Path

	// Reading

	var raw map[string]interface{}
	fetched, err := req.Storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Response

	if fetched == nil {
		return errNotFoundResponse(req, key), nil
	}

	if err := jsonutil.DecodeJSON(fetched.Value, &raw); err != nil {
		return nil, errwrap.Wrapf("json decoding failed: {{err}}", err)
	}
	resp := &logical.Response{
		Data: raw,
	}

	return resp, nil
}

// nolint:unused
func (b *tenantBackend) handleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	key := req.Path
	b.Logger().Debug("handleList", "key", key)

	// Reading

	ids, err := req.Storage.List(ctx, key)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}

	// Response

	// TODO the list can contain more data
	resp := &logical.Response{
		Data: map[string]interface{}{
			"ids": ids,
		},
	}

	return resp, nil
}

func validateTenantData(data *framework.FieldData) error {
	name, ok := data.GetOk("identifier")
	if !ok || len(name.(string)) == 0 {
		return fmt.Errorf("tenant identifier must not be empty")
	}
	return nil
}
