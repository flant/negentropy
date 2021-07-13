package backend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type roleBindingBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func roleBindingPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &roleBindingBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b roleBindingBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/role_binding",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"subjects": {
					Type:        framework.TypeSlice,
					Description: "Subjects list",
					Required:    true,
				},
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "TTL in seconds",
					Required:    true,
				},
				"require_mfa": {
					Type:        framework.TypeBool,
					Description: "Requires multi-factor authentication",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create roleBinding.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create roleBinding.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/role_binding/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a roleBinding",
					Required:    true,
				},
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"subjects": {
					Type:        framework.TypeSlice,
					Description: "Subjects list",
					Required:    true,
				},
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "TTL in seconds",
					Required:    true,
				},
				"require_mfa": {
					Type:        framework.TypeBool,
					Description: "Requires multi-factor authentication",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create roleBinding with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create roleBinding with preexistent ID.",
				},
			},
		},
		// Listing
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/role_binding/?",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all roleBindings IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/role_binding/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a roleBinding",
					Required:    true,
				},
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
					Required:    true,
				},
				"subjects": {
					Type:        framework.TypeSlice,
					Description: "Subjects list",
					Required:    true,
				},
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "TTL in seconds",
					Required:    true,
				},
				"require_mfa": {
					Type:        framework.TypeBool,
					Description: "Requires multi-factor authentication",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the role binding by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the role binding by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the role binding by ID.",
				},
			},
		},
	}
}

func (b *roleBindingBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)
		b.Logger().Debug("checking role binding existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		repo := model.NewRoleBindingRepository(tx)

		obj, err := repo.GetByID(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.TenantUUID == tenantID
		return exists, nil
	}
}

func (b *roleBindingBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := getCreationID(expectID, data)

		ttl := data.Get("ttl").(int)
		expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()

		subjects, err := parseSubjects(data)
		if err != nil {
			return nil, err
		}
		if len(subjects) == 0 {
			return responseErrMessage(req, "subjects must not be empty", http.StatusBadRequest)
		}

		roleBinding := &model.RoleBinding{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			ValidTill:  expiration,
			RequireMFA: data.Get("require_mfa").(bool),
			Subjects:   subjects,
			Origin:     model.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewRoleBindingRepository(tx)

		if err := repo.Create(roleBinding); err != nil {
			msg := "cannot create role binding"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"role_binding": roleBinding}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *roleBindingBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		ttl := data.Get("ttl").(int)
		expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()

		subjects, err := parseSubjects(data)
		if err != nil {
			return nil, err
		}
		if len(subjects) == 0 {
			return responseErrMessage(req, "subjects must not be empty", http.StatusBadRequest)
		}

		roleBinding := &model.RoleBinding{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Version:    data.Get("resource_version").(string),
			ValidTill:  expiration,
			RequireMFA: data.Get("require_mfa").(bool),
			Subjects:   subjects,
			Origin:     model.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		repo := model.NewRoleBindingRepository(tx)
		err = repo.Update(roleBinding)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"role_binding": roleBinding}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBindingBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewRoleBindingRepository(tx)

		err := repo.Delete(model.OriginIAM, id)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *roleBindingBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := model.NewRoleBindingRepository(tx)

		roleBinding, err := repo.GetByID(id)
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"role_binding": roleBinding}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBindingBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := model.NewRoleBindingRepository(tx)

		roleBindings, err := repo.List(tenantID)
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"role_bindings": roleBindings,
			},
		}
		return resp, nil
	}
}
