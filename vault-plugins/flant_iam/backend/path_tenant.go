package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

type tenantBackend struct {
	logical.Backend
	storage *memdb.MemDB
}

func tenantPaths(b logical.Backend, storage *memdb.MemDB) []*framework.Path {
	bb := &tenantBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b tenantBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			// using optional param in order to cover creation endpoint with empty id
			Pattern: "tenant" + uuid.OptionalPathParam("id"),
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

func (b *tenantBackend) handleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// creating or updating?

	id := data.Get("id").(string)
	isCreating := id == ""
	if isCreating {
		id = uuid.New()
	}

	tx := b.storage.Txn(true)
	defer tx.Abort()

	raw, err := tx.First(model.TenantType, model.TenantPK, id)
	if err != nil {
		return nil, err
	}

	var tenant *model.Tenant
	if isCreating {
		// we could have received the id
		if raw != nil {
			rr := logical.ErrorResponse("tenant already exists")
			return logical.RespondWithStatusCode(rr, req, http.StatusForbidden)
		}

		tenant = &model.Tenant{
			Id:         id,
			Identifier: data.Get("identifier").(string),
		}
	} else {
		if raw == nil {
			rr := logical.ErrorResponse("tenant not found")
			return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
		}
		tenant = raw.(*model.Tenant)
		tenant.Identifier = data.Get("identifier").(string)
	}

	// Validation

	// TODO: validation should depend on the storage
	//      validate field uniqueness
	//      validate resource_version
	// feature flags

	err = tx.Insert(model.TenantType, tenant)
	if err != nil {
		b.Logger().Debug("cannot save tenant", "err", err.Error())
		return logical.ErrorResponse("cannot save tenant"), nil
	}
	defer tx.Commit()

	// Response

	resp := &logical.Response{
		Data: map[string]interface{}{
			"id": id,
		},
	}

	successStatus := http.StatusOK
	if isCreating {
		successStatus++
	}
	return logical.RespondWithStatusCode(resp, req, successStatus)
}

func (b *tenantBackend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tx := b.storage.Txn(true)
	defer tx.Abort()

	// Verify existence

	id := data.Get("id").(string)
	raw, err := tx.First(model.TenantType, model.TenantPK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		rr := logical.ErrorResponse("tenant not found")
		return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
	}

	// Delete

	// FIXME: cascade deletion, e.g. deleteTenant()
	err = tx.Delete(model.TenantType, raw)
	if err != nil {
		return nil, err
	}
	defer tx.Commit()

	// Respond

	return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
}

func (b *tenantBackend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tx := b.storage.Txn(false)
	id := data.Get("id").(string)

	// Find

	raw, err := tx.First(model.TenantType, model.TenantPK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		rr := logical.ErrorResponse("tenant not found")
		return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
	}

	// Respond

	tenant := raw.(*model.Tenant)
	tenantJSON, err := tenant.Marshal(false)
	if err != nil {
		return nil, err
	}

	var responseData map[string]interface{}
	err = jsonutil.DecodeJSON(tenantJSON, &responseData)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: responseData,
	}

	return resp, nil
}

// nolint:unused
func (b *tenantBackend) handleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tx := b.storage.Txn(false)

	// Find

	iter, err := tx.Get(model.TenantType, model.TenantPK)
	if err != nil {
		return nil, err
	}

	tenants := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.Tenant)
		tenants = append(tenants, t.Id)
	}

	// Respond

	body, err := jsonutil.EncodeJSON(tenants)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"ids": body,
		},
	}

	return resp, nil
}
