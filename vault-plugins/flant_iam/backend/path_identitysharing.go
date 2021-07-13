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

type identitySharingBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func identitySharingPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &identitySharingBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b identitySharingBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/identity_sharing",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"destination_tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a destination tenant",
					Required:    true,
				},
				"groups": {
					Type:        framework.TypeStringSlice,
					Description: "ID of sharing groups",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate,
					Summary:  "Create identity sharing.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate,
					Summary:  "Create identity sharing.",
				},
			},
		},
		// Read, update, delete by uuid
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/identity_sharing/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of an identity sharing",
					Required:    true,
				},
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				// do we need it? Maybe we can change groups...
				// logical.UpdateOperation: &framework.PathOperation{
				// 	Callback: b.handleUpdate(),
				// 	Summary:  "Update the user by ID",
				// },
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead,
					Summary:  "Retrieve the identity sharing by ID",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete,
					Summary:  "Deletes the identity sharing by ID",
				},
			},
		},
		// Listing
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/identity_sharing/?",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList,
					Summary:  "Lists all tenant identity sharing IDs.",
				},
			},
		},
	}
}

func (b *identitySharingBackend) handleCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	sourceTenant := data.Get("tenant_uuid").(string)
	destTenant := data.Get("destination_tenant_uuid").(string)
	groups := data.Get("groups").([]string)

	tx := b.storage.Txn(true)
	defer tx.Abort()
	repo := model.NewIdentitySharingRepository(tx)

	is := &model.IdentitySharing{
		UUID:                  uuid.New(),
		SourceTenantUUID:      sourceTenant,
		DestinationTenantUUID: destTenant,
		Groups:                groups,
	}

	if err := repo.Create(is); err != nil {
		msg := "cannot create identity sharing"
		b.Logger().Debug(msg, "err", err.Error())
		return logical.ErrorResponse(msg), nil
	}

	if err := commit(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{"identity_sharing": is}}
	return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
}

func (b *identitySharingBackend) handleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	sourceTenant := data.Get("tenant_uuid").(string)

	tx := b.storage.Txn(false)
	defer tx.Abort()

	repo := model.NewIdentitySharingRepository(tx)

	list, err := repo.List(sourceTenant)
	if err != nil {
		return nil, err
	}

	_ = commit(tx, b.Logger())

	resp := &logical.Response{
		Data: map[string]interface{}{
			"identity_sharings": list,
		},
	}
	return resp, nil
}

func (b *identitySharingBackend) handleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(false)
	defer tx.Abort()

	repo := model.NewIdentitySharingRepository(tx)

	identitySharing, err := repo.GetByID(id)
	if err != nil {
		return responseErr(req, err)
	}
	_ = commit(tx, b.Logger())

	resp := &logical.Response{Data: map[string]interface{}{"identity_sharing": identitySharing}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *identitySharingBackend) handleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(true)
	defer tx.Abort()

	repo := model.NewIdentitySharingRepository(tx)

	err := repo.Delete(id)
	if err != nil {
		return responseErr(req, err)
	}
	if err := commit(tx, b.Logger()); err != nil {
		return nil, err
	}

	return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
}

func (b *identitySharingBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)
		b.Logger().Debug("checking identity sharing existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		defer tx.Abort()

		repo := model.NewIdentitySharingRepository(tx)

		obj, err := repo.GetByID(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.SourceTenantUUID == tenantID
		return exists, nil
	}
}
