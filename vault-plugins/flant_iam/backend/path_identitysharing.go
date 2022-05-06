package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
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
					Callback: b.handleCreate(false),
					Summary:  "Create identity sharing.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create identity sharing.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/identity_sharing/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
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
					Callback: b.handleCreate(true),
					Summary:  "Create identity sharing.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
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
				"groups": {
					Type:        framework.TypeStringSlice,
					Description: "ID of sharing groups",
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
					Callback: b.handleUpdate,
					Summary:  "Update  the identity sharing, allowed only change groups",
				},
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
		// List
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/identity_sharing/?",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived identity_sharings",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList,
					Summary:  "Lists all tenant identity sharing IDs.",
				},
			},
		},
	}
}

func (b *identitySharingBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request,
		data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create identity_sharing", "path", req.Path)
		id, err := backentutils.GetCreationID(expectID, data)
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}

		sourceTenant := data.Get("tenant_uuid").(string)
		destTenant := data.Get("destination_tenant_uuid").(string)
		groups := data.Get("groups").([]string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		is := &model.IdentitySharing{
			UUID:                  id,
			SourceTenantUUID:      sourceTenant,
			DestinationTenantUUID: destTenant,
			Groups:                groups,
		}

		denormalized, err := usecase.IdentityShares(tx, consts.OriginIAM).Create(is)
		if err != nil {
			err = fmt.Errorf("cannot create identity sharing:%w", err)
			b.Logger().Error(err.Error())
			return backentutils.ResponseErr(req, err)
		}

		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"identity_sharing": denormalized}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *identitySharingBackend) handleUpdate(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("update identity_sharing", "path", req.Path)
	id, err := backentutils.GetCreationID(true, data)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
	}

	tx := b.storage.Txn(true)
	defer tx.Abort()

	is := &model.IdentitySharing{
		UUID:             id,
		SourceTenantUUID: data.Get("tenant_uuid").(string),
		Groups:           data.Get("groups").([]string),
		Version:          data.Get("resource_version").(string),
	}

	denormalized, err := usecase.IdentityShares(tx, consts.OriginIAM).Update(is)
	if err != nil {
		err = fmt.Errorf("cannot update identity sharing:%w", err)
		b.Logger().Error(err.Error())
		return backentutils.ResponseErr(req, err)
	}

	if err = io.CommitWithLog(tx, b.Logger()); err != nil {
		return backentutils.ResponseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{"identity_sharing": denormalized}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *identitySharingBackend) handleList(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("list identity_sharings", "path", req.Path)
	var showArchived bool
	rawShowArchived, ok := data.GetOk("show_archived")
	if ok {
		showArchived = rawShowArchived.(bool)
	}
	sourceTenant := data.Get("tenant_uuid").(string)

	tx := b.storage.Txn(false)
	defer tx.Abort()

	list, err := usecase.IdentityShares(tx, consts.OriginIAM).List(sourceTenant, showArchived)
	if err != nil {
		b.Logger().Error(err.Error())
		return backentutils.ResponseErr(req, err)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"identity_sharings": list,
		},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *identitySharingBackend) handleRead(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("read identity_sharing", "path", req.Path)
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(false)
	defer tx.Abort()

	identitySharing, err := usecase.IdentityShares(tx, consts.OriginIAM).GetByID(id)
	if err != nil {
		b.Logger().Error(err.Error())
		return backentutils.ResponseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{"identity_sharing": identitySharing}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *identitySharingBackend) handleDelete(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("delete identity_sharing", "path", req.Path)
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(true)
	defer tx.Abort()

	err := usecase.IdentityShares(tx, consts.OriginIAM).Delete(id)
	if err != nil {
		err = fmt.Errorf("cannot delete identity sharing:%w", err)
		return backentutils.ResponseErr(req, err)
	}
	if err = io.CommitWithLog(tx, b.Logger()); err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
}

func (b *identitySharingBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)
		b.Logger().Debug("checking identity sharing existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		defer tx.Abort()

		repo := iam_repo.NewIdentitySharingRepository(tx)

		obj, err := repo.GetByID(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.SourceTenantUUID == tenantID
		return exists, nil
	}
}
