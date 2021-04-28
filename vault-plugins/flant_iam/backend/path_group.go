package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/io"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

type groupBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func groupPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &groupBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b groupBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/group",
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
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create group.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create group.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/group/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a group",
					Required:    true,
				},
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
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create group with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create group with preexistent ID.",
				},
			},
		},
		// Listing
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/group/?",
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
					Summary:  "Lists all groups IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{

			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/group/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a group",
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
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the service account by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the service account by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the service account by ID.",
				},
			},
		},
	}
}

func (b *groupBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)
		b.Logger().Debug("checking group existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		repo := NewGroupRepository(tx)

		obj, err := repo.GetById(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.TenantUUID == tenantID
		return exists, nil
	}
}

func (b *groupBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := getCreationID(expectID, data)

		group := &model.Group{
			UUID:            id,
			TenantUUID:      data.Get(model.TenantForeignPK).(string),
			BuiltinType:     "",
			Identifier:      data.Get("identifier").(string),
			Users:           data.Get("users").([]string),
			Groups:          data.Get("groups").([]string),
			ServiceAccounts: data.Get("service_accounts").([]string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewGroupRepository(tx)

		if err := repo.Create(group); err != nil {
			msg := "cannot create service account"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, group, http.StatusCreated)
	}
}

func (b *groupBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		group := &model.Group{
			UUID:            id,
			TenantUUID:      data.Get(model.TenantForeignPK).(string),
			Version:         data.Get("resource_version").(string),
			Identifier:      data.Get("identifier").(string),
			BuiltinType:     "",
			Users:           data.Get("users").([]string),
			Groups:          data.Get("groups").([]string),
			ServiceAccounts: data.Get("service_accounts").([]string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		repo := NewGroupRepository(tx)
		err := repo.Update(group)
		if err == ErrNotFound {
			return responseNotFound(req, model.GroupType)
		}
		if err == ErrVersionMismatch {
			return responseVersionMismatch(req)
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, group, http.StatusOK)
	}
}

func (b *groupBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewGroupRepository(tx)

		err := repo.Delete(id)
		if err == ErrNotFound {
			return responseNotFound(req, "service account not found")
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *groupBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := NewGroupRepository(tx)

		group, err := repo.GetById(id)
		if err == ErrNotFound {
			return responseNotFound(req, model.GroupType)
		}
		if err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, group, http.StatusOK)
	}
}

func (b *groupBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := NewGroupRepository(tx)

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

type GroupRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewGroupRepository(tx *io.MemoryStoreTxn) *GroupRepository {
	return &GroupRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *GroupRepository) Create(group *model.Group) error {
	tenant, err := r.tenantRepo.GetById(group.TenantUUID)
	if err != nil {
		return err
	}

	if group.Version != "" {
		return ErrVersionMismatch
	}
	group.Version = model.NewResourceVersion()
	group.FullIdentifier = model.CalcGroupFullIdentifier(group, tenant)

	err = r.db.Insert(model.GroupType, group)
	if err != nil {
		return err
	}
	return nil
}

func (r *GroupRepository) GetById(id string) (*model.Group, error) {
	raw, err := r.db.First(model.GroupType, model.ID, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	group := raw.(*model.Group)
	return group, nil
}

func (r *GroupRepository) Update(group *model.Group) error {
	stored, err := r.GetById(group.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != group.TenantUUID {
		return ErrNotFound
	}
	if stored.Version != group.Version {
		return ErrVersionMismatch
	}
	group.Version = model.NewResourceVersion()

	// Update

	tenant, err := r.tenantRepo.GetById(group.TenantUUID)
	if err != nil {
		return err
	}
	group.FullIdentifier = model.CalcGroupFullIdentifier(group, tenant)

	err = r.db.Insert(model.GroupType, group)
	if err != nil {
		return err
	}

	return nil
}

/*
TODO Clean from everywhere:
	* other groups
	* role_bindings
	* approvals
	* identity_sharings
*/
func (r *GroupRepository) Delete(id string) error {
	group, err := r.GetById(id)
	if err != nil {
		return err
	}

	return r.db.Delete(model.GroupType, group)
}

func (r *GroupRepository) List(tenantID string) ([]string, error) {
	iter, err := r.db.Get(model.GroupType, model.TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*model.Group)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}

func (r *GroupRepository) DeleteByTenant(tenantUUID string) error {
	_, err := r.db.DeleteAll(model.GroupType, model.TenantForeignPK, tenantUUID)
	return err
}
