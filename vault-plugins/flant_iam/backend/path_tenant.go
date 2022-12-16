package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
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
	paths := append(bb.paths(), bb.featureFlagPath())
	return paths
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
				"language": {
					Type:        framework.TypeString,
					Description: "preferred language",
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
				"language": {
					Type:        framework.TypeString,
					Description: "preferred language",
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
			Fields: map[string]*framework.FieldSchema{
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived tenants",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
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
				"language": {
					Type:        framework.TypeString,
					Description: "preferred language",
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
		// Erase by uuid
		{
			Pattern: "tenant/" + uuid.Pattern("uuid") + "/erase$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleErase(),
					Summary:  "Erase all of tenant by ID .",
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
		// Restore
		{
			Pattern: "tenant/" + uuid.Pattern("uuid") + "/restore" + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"full_restore": {
					Type:        framework.TypeBool,
					Description: "Option to restore full tenant data",
					Required:    false,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleRestore(),
					Summary:  "Restore the tenant by ID.",
				},
			},
		},

		// Feature flag for tenant
		b.featureFlagPath(),
	}
}

func (b *tenantBackend) handleListAvailableRoles() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		defer tx.Abort()

		available, err := usecase.TenantFeatureFlags(tx, tenantID).AvailableRoles()
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		if err := io.CommitWithLog(tx, b.Logger()); err != nil {
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

		t, err := usecase.Tenants(tx, consts.OriginIAM).GetByID(id)
		if err != nil {
			return false, err
		}
		return t != nil, nil
	}
}

func (b *tenantBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create tenant", "path", req.Path)
		id, err := backentutils.GetCreationID(expectID, data)
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}

		tenant := &model.Tenant{
			UUID:       id,
			Identifier: data.Get("identifier").(string),
			Language:   data.Get("language").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err = usecase.Tenants(tx, consts.OriginIAM).Create(tenant); err != nil {
			msg := "cannot create tenant"
			b.Logger().Error(msg, "err", err.Error())
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"tenant": tenant}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *tenantBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("update tenant", "path", req.Path)
		id := data.Get("uuid").(string)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		tenant := &model.Tenant{
			UUID:       id,
			Identifier: data.Get("identifier").(string),
			Version:    data.Get("resource_version").(string),
			Language:   data.Get("language").(string),
		}

		err := usecase.Tenants(tx, consts.OriginIAM).Update(tenant)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"tenant": tenant}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *tenantBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete tenant", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		id := data.Get("uuid").(string)

		err := usecase.Tenants(tx, consts.OriginIAM).Delete(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *tenantBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read tenant", "path", req.Path)
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)

		tenant, err := usecase.Tenants(tx, consts.OriginIAM).GetByID(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"tenant":       tenant,
			"full_restore": false, // TODO check if full restore available
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *tenantBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("listing tenants", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}

		tx := b.storage.Txn(false)
		tenants, err := usecase.Tenants(tx, consts.OriginIAM).List(showArchived)
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"tenants": tenants,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *tenantBackend) handleRestore() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("restore tenant", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		id := data.Get("uuid").(string)
		var fullRestore bool
		rawFullRestore, ok := data.GetOk("show_archived")
		if ok {
			fullRestore = rawFullRestore.(bool)
		}

		tenant, err := usecase.Tenants(tx, consts.OriginIAM).Restore(id, fullRestore)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"tenant": tenant,
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *tenantBackend) handleErase() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("erase tenant", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		id := data.Get("uuid").(string)

		err := usecase.Tenants(tx, consts.OriginIAM).CascadeErase(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}
