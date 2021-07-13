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
		// available roles based on FF
		{
			Pattern: "tenant/" + uuid.Pattern("uuid") + "/available_roles",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleListAvailableRoles(),
					Summary:  "Retrieve the tenant roles.",
				},
			},
		},

		// Feature flag for tenant
		b.featureFlagPath(),
	}
}

func (b *tenantBackend) handleListAvailableRoles() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		defer tx.Abort()
		repo := model.NewTenantFeatureFlagRepository(tx)

		available, err := repo.AvailableRoles(id)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		type cutRole struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}

		result := make([]cutRole, 0, len(available))

		for _, role := range available {
			result = append(result, cutRole{
				Name:        role.Name,
				Description: role.Description,
			})
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"available_roles": result,
			},
		}
		return resp, nil
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
		repo := model.NewTenantRepository(tx)

		t, err := repo.GetByID(id)
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
		repo := model.NewTenantRepository(tx)

		if err := repo.Create(tenant); err != nil {
			msg := "cannot create tenant"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := Commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"tenant": tenant}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
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

		repo := model.NewTenantRepository(tx)
		err := repo.Update(tenant)
		if err != nil {
			return ResponseErr(req, err)
		}
		if err := Commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"tenant": tenant}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *tenantBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewTenantRepository(tx)

		id := data.Get("uuid").(string)
		err := repo.Delete(id)
		if err != nil {
			return ResponseErr(req, err)
		}
		if err := Commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *tenantBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := model.NewTenantRepository(tx)

		tenant, err := repo.GetByID(id)
		if err != nil {
			return ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"tenant": tenant}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *tenantBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		repo := model.NewTenantRepository(tx)

		tenants, err := repo.List()
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"tenants": tenants,
			},
		}
		return resp, nil
	}
}
