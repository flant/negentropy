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
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
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
		b.Logger().Debug("checking tenant existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		repo := NewTenantRepository(tx)

		t, err := repo.GetById(id)
		if err != nil {
			return false, err
		}
		return t != nil, nil
	}
}

func (b *tenantBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := getCreationID(expectID, data)
		tenant := &model.Tenant{
			UUID:       id,
			Identifier: data.Get("identifier").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewTenantRepository(tx)

		if err := repo.Create(tenant); err != nil {
			msg := "cannot create tenant"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, tenant, http.StatusCreated)
	}
}

func (b *tenantBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		tenant := &model.Tenant{
			UUID:       id,
			Identifier: data.Get("identifier").(string),
			Version:    data.Get("resource_version").(string),
		}

		repo := NewTenantRepository(tx)
		err := repo.Update(tenant)
		if err == ErrNotFound {
			return responseNotFound(req, model.TenantType)
		}
		if err == ErrVersionMismatch {
			return responseVersionMismatch(req)
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, tenant, http.StatusOK)
	}
}

func (b *tenantBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewTenantRepository(tx)

		id := data.Get("uuid").(string)
		err := repo.Delete(id)
		if err == ErrNotFound {
			return responseNotFound(req, "tenant not found")
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *tenantBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := NewTenantRepository(tx)

		tenant, err := repo.GetById(id)
		if err == ErrNotFound {
			return responseNotFound(req, model.TenantType)
		}
		if err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, tenant, http.StatusOK)
	}
}

// nolint:unused
func (b *tenantBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		repo := NewTenantRepository(tx)

		list, err := repo.List()
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuids": list,
			},
		}
		return resp, nil
	}
}

type TenantRepository struct {
	db *memdb.Txn // called "db" not to provoke transaction semantics
}

func NewTenantRepository(tx *memdb.Txn) *TenantRepository {
	return &TenantRepository{tx}
}

func (r TenantRepository) Create(t *model.Tenant) error {
	t.Version = model.NewResourceVersion()
	return r.db.Insert(model.TenantType, t)
}

func (r TenantRepository) GetById(id string) (*model.Tenant, error) {
	raw, err := r.db.First(model.TenantType, model.ID, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*model.Tenant), nil
}

func (r TenantRepository) Update(updated *model.Tenant) error {
	stored, err := r.GetById(updated.UUID)
	if err != nil {
		return err
	}

	// Validate

	if stored.Version != updated.Version {
		return ErrVersionMismatch
	}
	updated.Version = model.NewResourceVersion()

	// Update

	return r.db.Insert(model.TenantType, updated)
}

func (r TenantRepository) Delete(id string) error {
	tenant, err := r.GetById(id)
	if err != nil {
		return err
	}

	return r.db.Delete(model.TenantType, tenant)
}

func (r TenantRepository) List() ([]string, error) {
	iter, err := r.db.Get(model.TenantType, model.ID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.Tenant)
		ids = append(ids, t.UUID)
	}
	return ids, nil
}
