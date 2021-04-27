package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type tenantBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func tenantPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &tenantBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b tenantBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "tenant",
			Fields: map[string]*framework.FieldSchema{
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create tenant.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create tenant.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "tenant/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create tenant with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create tenant with preexistent ID.",
				},
			},
		},
		// List
		{
			Pattern: "tenant/?",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all tenants IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{

			Pattern: "tenant/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the tenant by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the tenant by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the tenant by ID.",
				},
			},
		},
	}
}

func (b *tenantBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		b.Logger().Debug("checking tenant existence", "path", req.Path, "id", id)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}
		tx := b.storage.Txn(false)

		raw, err := tx.First(model.TenantType, model.ID, id)
		if err != nil {
			return false, err
		}

		return raw != nil, nil
	}
}

func (b *tenantBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		var id string

		if expectID {
			// for privileged access
			id = data.Get("uuid").(string)
		}

		if id == "" {
			id = uuid.New()
		}

		tenant := &model.Tenant{
			UUID:       id,
			Identifier: data.Get("identifier").(string),
		}

		// Validation

		// TODO: validation should depend on the storage
		//      validate field uniqueness
		//      validate resource_version
		// feature flags

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := tx.Insert(model.TenantType, tenant)
		if err != nil {
			msg := "cannot create tenant"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		defer tx.Commit()

		// Response

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuid": id,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *tenantBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		raw, err := tx.First(model.TenantType, model.ID, id)
		if err != nil {
			return nil, err
		}
		if raw == nil {
			rr := logical.ErrorResponse("tenant not found")
			return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
		}

		tenant := raw.(*model.Tenant)
		tenant.Identifier = data.Get("identifier").(string)

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
				"uuid": id,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *tenantBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(true)
		defer tx.Abort()

		// Verify existence

		id := data.Get("uuid").(string)
		raw, err := tx.First(model.TenantType, model.ID, id)
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
}

func (b *tenantBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		id := data.Get("uuid").(string)

		// Find

		raw, err := tx.First(model.TenantType, model.ID, id)
		if err != nil {
			return nil, err
		}
		if raw == nil {
			rr := logical.ErrorResponse("tenant not found")
			return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
		}

		// Respond

		return responseWithData(raw.(*model.Tenant))
	}
}

// nolint:unused
func (b *tenantBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)

		// Find

		iter, err := tx.Get(model.TenantType, model.ID)
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
			tenants = append(tenants, t.UUID)
		}

		// Respond

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuids": tenants,
			},
		}

		return resp, nil
	}
}
