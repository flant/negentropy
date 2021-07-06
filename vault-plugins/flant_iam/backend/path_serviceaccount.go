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
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "CIDRs",
					Required:    true,
				},
				"token_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
				"token_max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
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
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "CIDRs",
					Required:    true,
				},
				"token_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
				"token_max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
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
				"allowed_cidrs": {
					Type:        framework.TypeCommaStringSlice,
					Description: "CIDRs",
					Required:    true,
				},
				"token_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
					Required:    true,
				},
				"token_max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Multipass TTL in seconds",
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
		repo := model.NewServiceAccountRepository(tx)

		obj, err := repo.GetByID(id)
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
		ttl := data.Get("token_ttl").(int)
		maxttl := data.Get("token_max_ttl").(int)

		serviceAccount := &model.ServiceAccount{
			UUID:        id,
			TenantUUID:  data.Get(model.TenantForeignPK).(string),
			BuiltinType: "",
			Identifier:  data.Get("identifier").(string),
			CIDRs:       data.Get("allowed_cidrs").([]string),
			TokenTTL:    time.Duration(ttl),
			TokenMaxTTL: time.Duration(maxttl),
			Origin:      model.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewServiceAccountRepository(tx)

		if err := repo.Create(serviceAccount); err != nil {
			msg := "cannot create service account"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, serviceAccount, http.StatusCreated)
	}
}

func (b *serviceAccountBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)
		ttl := data.Get("token_ttl").(int)
		maxttl := data.Get("token_max_ttl").(int)

		serviceAccount := &model.ServiceAccount{
			UUID:        id,
			TenantUUID:  data.Get(model.TenantForeignPK).(string),
			Version:     data.Get("resource_version").(string),
			Identifier:  data.Get("identifier").(string),
			BuiltinType: "",
			CIDRs:       data.Get("allowed_cidrs").([]string),
			TokenTTL:    time.Duration(ttl),
			TokenMaxTTL: time.Duration(maxttl),
			Origin:      model.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		repo := model.NewServiceAccountRepository(tx)
		err := repo.Update(serviceAccount)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, serviceAccount, http.StatusOK)
	}
}

func (b *serviceAccountBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewServiceAccountRepository(tx)

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

func (b *serviceAccountBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := model.NewServiceAccountRepository(tx)

		serviceAccount, err := repo.GetByID(id)
		if err != nil {
			return responseErr(req, err)
		}

		return responseWithDataAndCode(req, serviceAccount, http.StatusOK)
	}
}

func (b *serviceAccountBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := model.NewServiceAccountRepository(tx)

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
