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

type serviceAccountBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func serviceAccountPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &serviceAccountBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b serviceAccountBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account",
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
					Summary:  "Create serviceAccount.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create serviceAccount.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a serviceAccount",
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
					Summary:  "Create serviceAccount with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create serviceAccount with preexistent ID.",
				},
			},
		},
		// Listing
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/?",
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
					Summary:  "Lists all serviceAccounts IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{

			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/service_account/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a serviceAccount",
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

func (b *serviceAccountBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)
		b.Logger().Debug("checking serviceAccount existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		repo := NewServiceAccountRepository(tx)

		obj, err := repo.GetById(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.TenantUUID == tenantID
		return exists, nil
	}
}

func (b *serviceAccountBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := getCreationID(expectID, data)
		serviceAccount := &model.ServiceAccount{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Identifier: data.Get("identifier").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewServiceAccountRepository(tx)

		if err := repo.Create(serviceAccount); err != nil {
			msg := "cannot create serviceAccount"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, serviceAccount, http.StatusCreated)
	}
}

func (b *serviceAccountBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		serviceAccount := &model.ServiceAccount{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Version:    data.Get("resource_version").(string),
			Identifier: data.Get("identifier").(string),
		}

		repo := NewServiceAccountRepository(tx)
		err := repo.Update(serviceAccount)
		if err == ErrNotFound {
			return responseNotFound(req, model.ServiceAccountType)
		}
		if err == ErrVersionMismatch {
			return responseVersionMismatch(req)
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, serviceAccount, http.StatusOK)
	}
}

func (b *serviceAccountBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewServiceAccountRepository(tx)

		err := repo.Delete(id)
		if err == ErrNotFound {
			return responseNotFound(req, "serviceAccount not found")
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *serviceAccountBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := NewServiceAccountRepository(tx)

		serviceAccount, err := repo.GetById(id)
		if err == ErrNotFound {
			return responseNotFound(req, model.ServiceAccountType)
		}
		if err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, serviceAccount, http.StatusOK)
	}
}

func (b *serviceAccountBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := NewServiceAccountRepository(tx)

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

type ServiceAccountRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewServiceAccountRepository(tx *io.MemoryStoreTxn) *ServiceAccountRepository {
	return &ServiceAccountRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *ServiceAccountRepository) Create(serviceAccount *model.ServiceAccount) error {
	tenant, err := r.tenantRepo.GetById(serviceAccount.TenantUUID)
	if err != nil {
		return err
	}

	serviceAccount.Version = model.NewResourceVersion()
	serviceAccount.FullIdentifier = serviceAccount.Identifier + "@" + tenant.Identifier

	err = r.db.Insert(model.ServiceAccountType, serviceAccount)
	if err != nil {
		return err
	}
	return nil
}

func (r *ServiceAccountRepository) GetById(id string) (*model.ServiceAccount, error) {
	raw, err := r.db.First(model.ServiceAccountType, model.ID, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	serviceAccount := raw.(*model.ServiceAccount)
	return serviceAccount, nil
}

func (r *ServiceAccountRepository) Update(serviceAccount *model.ServiceAccount) error {
	stored, err := r.GetById(serviceAccount.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != serviceAccount.TenantUUID {
		return ErrNotFound
	}
	if stored.Version != serviceAccount.Version {
		return ErrVersionMismatch
	}
	serviceAccount.Version = model.NewResourceVersion()

	// Update

	tenant, err := r.tenantRepo.GetById(serviceAccount.TenantUUID)
	if err != nil {
		return err
	}
	serviceAccount.FullIdentifier = serviceAccount.Identifier + "@" + tenant.Identifier

	err = r.db.Insert(model.ServiceAccountType, serviceAccount)
	if err != nil {
		return err
	}

	return nil
}

func (r *ServiceAccountRepository) Delete(id string) error {
	serviceAccount, err := r.GetById(id)
	if err != nil {
		return err
	}

	return r.db.Delete(model.ServiceAccountType, serviceAccount)
}

func (r *ServiceAccountRepository) List(tenantID string) ([]string, error) {
	iter, err := r.db.Get(model.ServiceAccountType, model.TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*model.ServiceAccount)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}

func (r *ServiceAccountRepository) DeleteByTenant(tenantUUID string) error {
	_, err := r.db.DeleteAll(model.ServiceAccountType, model.TenantForeignPK, tenantUUID)
	return err
}
