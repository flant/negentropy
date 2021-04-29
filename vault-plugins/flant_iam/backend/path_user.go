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

type userBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func userPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &userBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b userBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user",
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
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create user.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create user.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a user",
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
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create user with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create user with preexistent ID.",
				},
			},
		},
		// Listing
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user/?",
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
					Summary:  "Lists all users IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{

			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a user",
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
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the user by ID",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the user by ID",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the user by ID",
				},
			},
		},
		// Multipass creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user/" + uuid.Pattern("owner_uuid") + "/multipass",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of an owner",
					Required:    true,
				},
				"ttl": {
					Type:        framework.TypeInt,
					Description: "TTL in seconds",
					Required:    true,
				},
				"max_ttl": {
					Type:        framework.TypeInt,
					Description: "Max TTL in seconds",
					Required:    true,
				},
				"description": {
					Type:        framework.TypeString,
					Description: "The purpose of issuing",
					Required:    true,
				},
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Allowed CIDRs to use the multipass from",
					Required:    true,
				},
				"allowed_roles": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Allowed roles to use the multipass with",
					Required:    true,
				},
			},
			ExistenceCheck: neverExisting,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleMultipassCreate(),
					Summary:  "Create user multipass.",
				},
			},
		},
		// Multipass read or delete
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user/" + uuid.Pattern("owner_uuid") + "/multipass/" + uuid.Pattern("uuid"),
			Fields: map[string]*framework.FieldSchema{

				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a owner",
					Required:    true,
				},
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a multipass",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleMultipassRead(),
					Summary:  "Get multipass by ID",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleMultipassDelete(),
					Summary:  "Delete multipass by ID",
				},
			},
		},
		// Multipass list
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user/" + uuid.Pattern("owner_uuid") + "/multipass/?",
			Fields: map[string]*framework.FieldSchema{

				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"owner_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of an owner",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleMultipassList(),
					Summary:  "List multipass IDs",
				},
			},
		},
	}
}

// neverExisting  is a useful existence check handler to always trigger create operation
func neverExisting(context.Context, *logical.Request, *framework.FieldData) (bool, error) {
	return false, nil
}

func (b *userBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)
		b.Logger().Debug("checking user existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		repo := model.NewUserRepository(tx)

		obj, err := repo.GetByID(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.TenantUUID == tenantID
		return exists, nil
	}
}

func (b *userBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := getCreationID(expectID, data)
		user := &model.User{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Identifier: data.Get("identifier").(string),
			Origin:     model.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewUserRepository(tx)

		if err := repo.Create(user); err != nil {
			msg := "cannot create user"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, user, http.StatusCreated)
	}
}

func (b *userBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		user := &model.User{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Version:    data.Get("resource_version").(string),
			Identifier: data.Get("identifier").(string),
			Origin:     model.OriginIAM,
		}

		repo := model.NewUserRepository(tx)
		err := repo.Update(user)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, user, http.StatusOK)
	}
}

func (b *userBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewUserRepository(tx)

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

func (b *userBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := model.NewUserRepository(tx)

		user, err := repo.GetById(id)
		if err != nil {
			return responseErr(req, err)
		}

		return responseWithDataAndCode(req, user, http.StatusOK)
	}
}

func (b *userBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := model.NewUserRepository(tx)

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

func (b *userBackend) handleMultipassCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		var (
			ttl       = time.Duration(data.Get("ttl").(int)) * time.Second
			maxTTL    = time.Duration(data.Get("max_ttl").(int)) * time.Second
			validTill = time.Now().Add(ttl).Unix()
		)

		multipass := &model.Multipass{
			UUID:        uuid.New(),
			TenantUUID:  data.Get("tenant_uuid").(string),
			OwnerUUID:   data.Get("owner_uuid").(string),
			OwnerType:   model.MultipassOwnerUser,
			Description: data.Get("description").(string),
			TTL:         ttl,
			MaxTTL:      maxTTL,
			ValidTill:   validTill,
			CIDRs:       data.Get("allowed_cidrs").([]string),
			Roles:       data.Get("allowed_roles").([]string),
		}

		tx := b.storage.Txn(true)
		repo := NewMultipassRepository(tx)

		err := repo.Create(multipass)
		if err == ErrNotFound {
			return responseNotFound(req, "something in the path")
		}
		if err != nil {
			return nil, err
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, multipass, http.StatusCreated)
	}
}

func (b *userBackend) handleMultipassDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		filter := &model.Multipass{
			UUID:       data.Get("uuid").(string),
			TenantUUID: data.Get("tenant_uuid").(string),
			OwnerUUID:  data.Get("owner_uuid").(string),
			OwnerType:  model.MultipassOwnerUser,
		}

		tx := b.storage.Txn(true)
		repo := NewMultipassRepository(tx)

		err := repo.Delete(filter)
		if err != ErrNotFound {
			return responseNotFound(req, "something in the path")
		}
		if err != nil {
			return nil, err
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func (b *userBackend) handleMultipassRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		filter := &model.Multipass{
			UUID:       data.Get("uuid").(string),
			TenantUUID: data.Get("tenant_uuid").(string),
			OwnerUUID:  data.Get("owner_uuid").(string),
			OwnerType:  model.MultipassOwnerUser,
		}

		tx := b.storage.Txn(false)
		repo := NewMultipassRepository(tx)

		mp, err := repo.Get(filter)
		if err != ErrNotFound {
			return responseNotFound(req, "something in the path")
		}
		if err != nil {
			return nil, err
		}

		return responseWithData(mp)
	}
}

func (b *userBackend) handleMultipassList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		filter := &model.Multipass{
			TenantUUID: data.Get("tenant_uuid").(string),
			OwnerUUID:  data.Get("owner_uuid").(string),
			OwnerType:  model.MultipassOwnerUser,
		}

		tx := b.storage.Txn(false)
		repo := NewMultipassRepository(tx)

		ids, err := repo.List(filter)
		if err != ErrNotFound {
			return responseNotFound(req, "something in the path")
		}
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuids": ids,
			},
		}

		return resp, nil
	}
}

type MultipassRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewMultipassRepository(tx *io.MemoryStoreTxn) *MultipassRepository {
	return &MultipassRepository{db: tx}
}

func (r *MultipassRepository) validate(multipass *model.Multipass) error {
	tenantRepo := NewTenantRepository(r.db)
	_, err := tenantRepo.GetByID(multipass.TenantUUID)
	if err != nil {
		return err
	}

	if multipass.OwnerType == model.MultipassOwnerUser {
		repo := NewUserRepository(r.db)
		owner, err := repo.GetByID(multipass.OwnerUUID)
		if err != nil {
			return err
		}
		if owner.TenantUUID != multipass.TenantUUID {
			return ErrNotFound
		}
	}

	if multipass.OwnerType == model.MultipassOwnerServiceAccount {
		repo := NewServiceAccountRepository(r.db)
		owner, err := repo.GetByID(multipass.OwnerUUID)
		if err != nil {
			return err
		}
		if owner.TenantUUID != multipass.TenantUUID {
			return ErrNotFound
		}
	}

	return nil
}

func (r *MultipassRepository) Create(mp *model.Multipass) error {
	err := r.validate(mp)
	if err != nil {
		return err
	}

	return r.db.Insert(model.MultipassType, mp)
}

func (r *MultipassRepository) Delete(filter *model.Multipass) error {
	err := r.validate(filter)
	if err != nil {
		return err
	}

	return r.db.Delete(model.MultipassType, filter)
}

func (r *MultipassRepository) Get(filter *model.Multipass) (*model.Multipass, error) {
	err := r.validate(filter)
	if err != nil {
		return nil, err
	}

	return r.GetByID(filter.UUID)
}

func (r *MultipassRepository) GetByID(id string) (*model.Multipass, error) {
	raw, err := r.db.First(model.MultipassType, model.PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	multipass := raw.(*model.Multipass)
	return multipass, nil
}

func (r *MultipassRepository) List(filter *model.Multipass) ([]string, error) {
	err := r.validate(filter)
	if err != nil {
		return nil, err
	}

	iter, err := r.db.Get(model.MultipassType, model.OwnerForeignPK, filter.OwnerUUID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		mp := raw.(*model.Multipass)
		ids = append(ids, mp.UUID)
	}
	return ids, nil
}

type UserRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewUserRepository(tx *io.MemoryStoreTxn) *UserRepository {
	return &UserRepository{db: tx}
}

func (r *UserRepository) Create(user *model.User) error {
	tenantRepo := NewTenantRepository(r.db)
	tenant, err := tenantRepo.GetByID(user.TenantUUID)
	if err != nil {
		return err
	}

	user.Version = model.NewResourceVersion()
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier

	err = r.db.Insert(model.UserType, user)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(id string) (*model.User, error) {
	raw, err := r.db.First(model.UserType, model.PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	user := raw.(*model.User)
	return user, nil
}

func (r *UserRepository) Update(user *model.User) error {
	stored, err := r.GetByID(user.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != user.TenantUUID {
		return ErrNotFound
	}
	if stored.Version != user.Version {
		return ErrVersionMismatch
	}
	user.Version = model.NewResourceVersion()

	// Update

	tenantRepo := NewTenantRepository(r.db)
	tenant, err := tenantRepo.GetByID(user.TenantUUID)
	if err != nil {
		return err
	}
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier

	err = r.db.Insert(model.UserType, user)
	if err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) Delete(id string) error {
	user, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(model.UserType, user)
}

func (r *UserRepository) List(tenantID string) ([]string, error) {
	iter, err := r.db.Get(model.UserType, model.TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*model.User)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}

func (r *UserRepository) DeleteByTenant(tenantUUID string) error {
	_, err := r.db.DeleteAll(model.UserType, model.TenantForeignPK, tenantUUID)
	return err
}

func (r *UserRepository) Iter(action func(*model.User) (bool, error)) error {
	iter, err := r.db.Get(model.UserType, model.PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.User)
		next, err := action(t)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
}
