package backend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/io"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
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
				"users": {
					Type:        framework.TypeCommaStringSlice,
					Description: "User UUIDs",
					Required:    true,
				},
				"groups": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Group UUIDs",
					Required:    true,
				},
				"service_accounts": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Service account UUIDs",
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
				"users": {
					Type:        framework.TypeCommaStringSlice,
					Description: "User UUIDs",
					Required:    true,
				},
				"groups": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Group UUIDs",
					Required:    true,
				},
				"service_accounts": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Service account UUIDs",
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
				"users": {
					Type:        framework.TypeCommaStringSlice,
					Description: "User UUIDs",
					Required:    true,
				},
				"groups": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Group UUIDs",
					Required:    true,
				},
				"service_accounts": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Service account UUIDs",
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
		repo := NewRoleBindingRepository(tx)

		obj, err := repo.GetById(id)
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

		roleBinding := &model.RoleBinding{
			UUID:            id,
			TenantUUID:      data.Get(model.TenantForeignPK).(string),
			ValidTill:       expiration,
			RequireMFA:      data.Get("require_mfa").(bool),
			Users:           data.Get("users").([]string),
			Groups:          data.Get("groups").([]string),
			ServiceAccounts: data.Get("service_accounts").([]string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewRoleBindingRepository(tx)

		if err := repo.Create(roleBinding); err != nil {
			msg := "cannot create role binding"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, roleBinding, http.StatusCreated)
	}
}

func (b *roleBindingBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		ttl := data.Get("ttl").(int)
		expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()

		roleBinding := &model.RoleBinding{
			UUID:            id,
			TenantUUID:      data.Get(model.TenantForeignPK).(string),
			Version:         data.Get("resource_version").(string),
			ValidTill:       expiration,
			RequireMFA:      data.Get("require_mfa").(bool),
			Users:           data.Get("users").([]string),
			Groups:          data.Get("groups").([]string),
			ServiceAccounts: data.Get("service_accounts").([]string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		repo := NewRoleBindingRepository(tx)
		err := repo.Update(roleBinding)
		if err == ErrNotFound {
			return responseNotFound(req, model.RoleBindingType)
		}
		if err == ErrVersionMismatch {
			return responseVersionMismatch(req)
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, roleBinding, http.StatusOK)
	}
}

func (b *roleBindingBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewRoleBindingRepository(tx)

		err := repo.Delete(id)
		if err == ErrNotFound {
			return responseNotFound(req, "role binding not found")
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *roleBindingBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := NewRoleBindingRepository(tx)

		roleBinding, err := repo.GetById(id)
		if err == ErrNotFound {
			return responseNotFound(req, model.RoleBindingType)
		}
		if err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, roleBinding, http.StatusOK)
	}
}

func (b *roleBindingBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := NewRoleBindingRepository(tx)

		list, err := repo.List(tenantID)
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

type RoleBindingRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewRoleBindingRepository(tx *io.MemoryStoreTxn) *RoleBindingRepository {
	return &RoleBindingRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *RoleBindingRepository) Create(roleBinding *model.RoleBinding) error {
	_, err := r.tenantRepo.GetById(roleBinding.TenantUUID)
	if err != nil {
		return err
	}

	if roleBinding.Version != "" {
		return ErrVersionMismatch
	}
	roleBinding.Version = model.NewResourceVersion()

	err = r.db.Insert(model.RoleBindingType, roleBinding)
	if err != nil {
		return err
	}
	return nil
}

func (r *RoleBindingRepository) GetById(id string) (*model.RoleBinding, error) {
	raw, err := r.db.First(model.RoleBindingType, model.ID, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	roleBinding := raw.(*model.RoleBinding)
	return roleBinding, nil
}

func (r *RoleBindingRepository) Update(roleBinding *model.RoleBinding) error {
	stored, err := r.GetById(roleBinding.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != roleBinding.TenantUUID {
		return ErrNotFound
	}
	if stored.Version != roleBinding.Version {
		return ErrVersionMismatch
	}
	roleBinding.Version = model.NewResourceVersion()

	// Update

	err = r.db.Insert(model.RoleBindingType, roleBinding)
	if err != nil {
		return err
	}

	return nil
}

func (r *RoleBindingRepository) Delete(id string) error {
	roleBinding, err := r.GetById(id)
	if err != nil {
		return err
	}

	return r.db.Delete(model.RoleBindingType, roleBinding)
}

func (r *RoleBindingRepository) List(tenantID string) ([]string, error) {
	iter, err := r.db.Get(model.RoleBindingType, model.TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*model.RoleBinding)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}

func (r *RoleBindingRepository) DeleteByTenant(tenantUUID string) error {
	_, err := r.db.DeleteAll(model.RoleBindingType, model.TenantForeignPK, tenantUUID)
	return err
}
