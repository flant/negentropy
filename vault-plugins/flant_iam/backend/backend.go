package backend

import (
	"context"
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
	sender KafkaSender
}

func layerBackendPaths(b *framework.Backend, keyman *key.Manager, schema Schema, sender KafkaSender) []*framework.Path {
	bb := &layerBackend{
		Backend: b,
		keyman:  keyman,
		schema:  schema,
		sender:  sender,
	}
	return bb.paths()
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

func getKey(sch Schema, path, id string) (string, bool) {
	isCreating := id == ""
	if isCreating {
		key := path + "/" + sch.GenerateID()
		return key, isCreating
	}
	return path, isCreating
}

func (b *layerBackend) handleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// creating or updating?
	id := data.Get(b.keyman.IDField()).(string)
	key, isCreating := getKey(b.schema, req.Path, id)
	repo := Repository{b.schema}

	// Validation

	// TODO: validation should depend on the storage
	//      validate field uniqueness
	//      validate resource_version

	_, err := repo.Get(ctx, req.Storage, key)
	if err != nil && err != ErrNotFound {
		return nil, err
	}

	isUpdating := !isCreating
	if isUpdating && err == ErrNotFound {
		// nothing to update
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
	input, err := b.schema.ParseData(data)
	if err != nil {
		return nil, err
	}

	// TODO resource version
	// if stored.Version() > input.Version() {
	//	return nil, &logical.StatusBadRequest{Err: "input resource version is older than stored one"}
	// }

	err = repo.Put(ctx, req.Storage, key, input)
	if err != nil {
		return nil, err
	}

	message := &Message{
		Meta: Meta{
			Type: b.schema.Type(),
			Id:   data.Get(b.keyman.IDField()).(string),
			Key:  key,
		},
		Data: input,
	}

	err = b.sender.Send(ctx, message, b.schema.SyncTopics())
	if err != nil {
		b.Logger().Warn("cannot send data to broker", "key", key, "error", err)
	}

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

func (b *layerBackend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	key := req.Path
	repo := Repository{b.schema}
	b.Logger().Debug("handleDelete", "key", key)

	// Ensure it exists
	stored, err := repo.Get(ctx, req.Storage, key)

	if err == ErrNotFound {
		// nothing to update
		return errNotFoundResponse(req, key), nil
	}

	if err != nil {
		return nil, err
	}

	// Deletion

	// TODO: cascade deletion

	err = req.Storage.Delete(ctx, key)
	if err != nil {
		return nil, err
	}

	message := &Message{
		Meta: Meta{
			Type: b.schema.Type(),
			Id:   data.Get(b.keyman.IDField()).(string),
			Key:  key,
		},
		Data: stored,
	}

	// TODO: Real kafka
	err = b.sender.Delete(ctx, message, b.schema.SyncTopics())
	return nil, err
}

func (b *layerBackend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
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

func errNotFoundResponse(req *logical.Request, key string) *logical.Response {
	errResp := logical.ErrorResponse("Not found %v%v", req.MountPoint, key)
	resp, _ := logical.RespondWithStatusCode(errResp, req, http.StatusNotFound)
	return resp
}
