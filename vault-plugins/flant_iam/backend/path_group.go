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
				"subjects": {
					Type:        framework.TypeSlice,
					Description: "Subjects list",
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
				"subjects": {
					Type:        framework.TypeSlice,
					Description: "Subjects list",
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
				"subjects": {
					Type:        framework.TypeSlice,
					Description: "Subjects list",
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
		repo := model.NewGroupRepository(tx)

		obj, err := repo.GetByID(id)
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

		subjects, err := parseSubjects(data.Get("subjects"))
		if err != nil {
			return nil, err
		}
		if len(subjects) == 0 {
			return responseErrMessage(req, "subjects must not be empty", http.StatusBadRequest)
		}

		group := &model.Group{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Identifier: data.Get("identifier").(string),
			Subjects:   subjects,
			Origin:     model.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewGroupRepository(tx)

		if err := repo.Create(group); err != nil {
			msg := "cannot create service account"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"group": group}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *groupBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		subjects, err := parseSubjects(data.Get("subjects"))
		if err != nil {
			return nil, err
		}
		if len(subjects) == 0 {
			return responseErrMessage(req, "subjects must not be empty", http.StatusBadRequest)
		}

		group := &model.Group{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Version:    data.Get("resource_version").(string),
			Identifier: data.Get("identifier").(string),
			Subjects:   subjects,
			Origin:     model.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		repo := model.NewGroupRepository(tx)
		err = repo.Update(group)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"group": group}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *groupBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewGroupRepository(tx)

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

func (b *groupBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := model.NewGroupRepository(tx)
		group, err := repo.GetByID(id)
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"group": group}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *groupBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := model.NewGroupRepository(tx)

		groups, err := repo.List(tenantID)
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"groups": groups,
			},
		}
		return resp, nil
	}
}
