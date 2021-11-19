package backend

//goland:noinspection GoUnsortedImport
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
					Type:        framework.TypeString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"members": {
					Type:        framework.TypeSlice,
					Description: "Members list",
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
					Type:        framework.TypeString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"members": {
					Type:        framework.TypeSlice,
					Description: "Members list",
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
		// List
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/group/?",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived groups",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
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
					Type:        framework.TypeString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"members": {
					Type:        framework.TypeSlice,
					Description: "Members list",
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
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)
		b.Logger().Debug("checking group existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		repo := iam_repo.NewGroupRepository(tx)

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
		b.Logger().Debug("create group", "path", req.Path)
		var (
			id, err    = backentutils.GetCreationID(expectID, data)
			tenantUUID = data.Get(iam_repo.TenantForeignPK).(string)
		)

		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}

		members, err := parseMembers(data.Get("members"))
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}
		if len(members) == 0 {
			return backentutils.ResponseErrMessage(req, "members must not be empty", http.StatusBadRequest)
		}

		group := &model.Group{
			UUID:       id,
			TenantUUID: data.Get(iam_repo.TenantForeignPK).(string),
			Identifier: data.Get("identifier").(string),
			Members:    members,
			Origin:     consts.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err = usecase.Groups(tx, tenantUUID).Create(group); err != nil {
			msg := "cannot create group"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"group": group}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *groupBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("update group", "path", req.Path)
		var (
			id         = data.Get("uuid").(string)
			tenantUUID = data.Get(iam_repo.TenantForeignPK).(string)
		)

		members, err := parseMembers(data.Get("members"))
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}
		if len(members) == 0 {
			return backentutils.ResponseErrMessage(req, "members must not be empty", http.StatusBadRequest)
		}

		group := &model.Group{
			UUID:       id,
			TenantUUID: data.Get(iam_repo.TenantForeignPK).(string),
			Version:    data.Get("resource_version").(string),
			Identifier: data.Get("identifier").(string),
			Members:    members,
			Origin:     consts.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err = usecase.Groups(tx, tenantUUID).Update(group)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"group": group}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *groupBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete groups", "path", req.Path)
		var (
			id         = data.Get("uuid").(string)
			tenantUUID = data.Get(iam_repo.TenantForeignPK).(string)
		)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.Groups(tx, tenantUUID).Delete(consts.OriginIAM, id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *groupBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read group", "path", req.Path)
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := iam_repo.NewGroupRepository(tx)
		group, err := repo.GetByID(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"group": group}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *groupBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list groups", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := iam_repo.NewGroupRepository(tx)

		groups, err := repo.List(tenantID, showArchived)
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"groups": groups,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
