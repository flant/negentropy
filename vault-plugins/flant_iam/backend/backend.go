package backend

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/key"
)

type layerBackend struct {
	logical.Backend

	keyman *key.Manager
	schema Schema
}

func (b layerBackend) paths() []*framework.Path {
	fields := b.schema.Fields()

	// adding the field to make it accessible within handlers
	fields[b.keyman.IDField()] = &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: "ID of a " + b.keyman.EntryName(),
	}

	return []*framework.Path{
		{
			// using optional param in order to cover creation endpoint with empty id
			Pattern: b.keyman.EntryPattern(),
			Fields:  fields,
			Operations: map[logical.Operation]framework.OperationHandler{
				// POST, create or update
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleWrite,
					Summary:  "Update the " + b.keyman.EntryName() + " by ID.",
				},
				// GET
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead,
					Summary:  "Retrieve the " + b.keyman.EntryName() + " by ID.",
				},
				// DELETE
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete,
					Summary:  "Deletes the " + b.keyman.EntryName() + " by ID.",
				},
			},
		},

		{
			Pattern: b.keyman.ListPattern(),
			Operations: map[logical.Operation]framework.OperationHandler{
				// GET
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList,
					Summary:  "Lists all " + b.keyman.EntryName() + "s IDs.",
				},
			},
		},
	}
}

func (b *layerBackend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("handleRead", "path", req.Path)
	key := req.Path

	// Reading

	var rawData map[string]interface{}
	fetchedData, err := req.Storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Response

	if fetchedData == nil {
		return errNotFoundResponse(req, key), nil
	}

	if err := jsonutil.DecodeJSON(fetchedData.Value, &rawData); err != nil {
		return nil, errwrap.Wrapf("json decoding failed: {{err}}", err)
	}
	resp := &logical.Response{
		Data: rawData,
	}

	return resp, nil
}

// nolint:unused
func (b *layerBackend) handleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	key := req.Path
	b.Logger().Debug("handleList", "key", key)

	// Reading

	fetchedData, err := req.Storage.List(ctx, key)
	if err != nil {
		return nil, err
	}
	if fetchedData == nil {
		fetchedData = []string{}
	}

	// Response

	// TODO the list can contain more data
	resp := &logical.Response{
		Data: map[string]interface{}{
			"ids": fetchedData,
		},
	}

	return resp, nil
}

func (b *layerBackend) handleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// creating or updating?

	id := data.Get(b.keyman.IDField()).(string)
	successStatus := http.StatusOK
	isCreating := false
	key := req.Path

	if id == "" {
		// the creation here
		id = b.schema.GenerateID()
		key = req.Path + "/" + id
		successStatus = http.StatusCreated
		isCreating = true
		b.Logger().Debug("creating")
	}

	// Validation

	// TODO: validation should depend on the storage
	//      validate field uniqueness
	//      validate resource_version

	exists, err := checkExistence(ctx, req.Storage, key)
	if err != nil {
		return nil, err
	}
	if !exists && !isCreating {
		errResp := logical.ErrorResponse("No value at %v%v", req.MountPoint, key)
		resp, _ := logical.RespondWithStatusCode(errResp, req, 404)
		return resp, nil
	}

	b.Logger().Debug("writing", "key", key)

	err = b.schema.Validate(data)
	if err != nil {
		return nil, &logical.StatusBadRequest{Err: err.Error()}
	}

	// Storing

	buf, err := json.Marshal(req.Data)
	if err != nil {
		return nil, errwrap.Wrapf("json encoding failed: {{err}}", err)
	}

	entry := &logical.StorageEntry{
		Key:   key,
		Value: buf,
	}
	// TODO send to kafka
	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return nil, err
	}

	// Response

	resp := &logical.Response{
		Data: map[string]interface{}{
			"id": id,
		},
	}
	return logical.RespondWithStatusCode(resp, req, successStatus)
}

func (b *layerBackend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	key := req.Path
	b.Logger().Debug("handleDelete", "key", key)

	// Validation

	exists, err := checkExistence(ctx, req.Storage, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return errNotFoundResponse(req, key), nil
	}

	// Deletion

	// TODO: cascade deletion
	// TODO send to kafka about every deletion
	err = req.Storage.Delete(ctx, key)
	return nil, err
}

// checkExistence checks for the existence.
//
// DO NOT USE IT IN THE logical.Backend#ExistenceCheck! It does not comply with the key-value storage logic.
func checkExistence(ctx context.Context, storage logical.Storage, key string) (bool, error) {
	out, err := storage.Get(ctx, key)
	if err != nil {
		return false, errwrap.Wrapf("existence check failed: {{err}}", err)
	}
	return out != nil, nil
}

func errNotFoundResponse(req *logical.Request, key string) *logical.Response {
	errResp := logical.ErrorResponse("Not found %v%v", req.MountPoint, key)
	resp, _ := logical.RespondWithStatusCode(errResp, req, http.StatusNotFound)
	return resp
}
