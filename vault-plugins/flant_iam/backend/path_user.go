package backend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
)

type userBackend struct {
	logical.Backend
	storage         *io.MemoryStore
	tokenController *jwt.Controller
}

func userPaths(b logical.Backend, tokenController *jwt.Controller, storage *io.MemoryStore) []*framework.Path {
	bb := &userBackend{
		Backend:         b,
		storage:         storage,
		tokenController: tokenController,
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
				"first_name": {
					Type:        framework.TypeString,
					Description: "first_name",
					Required:    true,
				},
				"last_name": {
					Type:        framework.TypeString,
					Description: "last_name",
					Required:    true,
				},
				"display_name": {
					Type:        framework.TypeString,
					Description: "display_name",
					Required:    true,
				},
				"email": {
					Type:        framework.TypeString,
					Description: "email",
					Required:    true,
				},
				"additional_emails": {
					Type:        framework.TypeStringSlice,
					Description: "additional_emails",
					Required:    true,
				},
				"mobile_phone": {
					Type:        framework.TypeString,
					Description: "mobile_phone",
					Required:    true,
				},
				"additional_phones": {
					Type:        framework.TypeStringSlice,
					Description: "additional_phones",
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
				"first_name": {
					Type:        framework.TypeString,
					Description: "first_name",
					Required:    true,
				},
				"last_name": {
					Type:        framework.TypeString,
					Description: "last_name",
					Required:    true,
				},
				"display_name": {
					Type:        framework.TypeString,
					Description: "display_name",
					Required:    true,
				},
				"email": {
					Type:        framework.TypeString,
					Description: "email",
					Required:    true,
				},
				"additional_emails": {
					Type:        framework.TypeStringSlice,
					Description: "additional_emails",
					Required:    true,
				},
				"mobile_phone": {
					Type:        framework.TypeString,
					Description: "mobile_phone",
					Required:    true,
				},
				"additional_phones": {
					Type:        framework.TypeStringSlice,
					Description: "additional_phones",
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
		// List
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user/?",
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived users",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
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
				"first_name": {
					Type:        framework.TypeString,
					Description: "first_name",
					Required:    true,
				},
				"last_name": {
					Type:        framework.TypeString,
					Description: "last_name",
					Required:    true,
				},
				"display_name": {
					Type:        framework.TypeString,
					Description: "display_name",
					Required:    true,
				},
				"email": {
					Type:        framework.TypeString,
					Description: "email",
					Required:    true,
				},
				"additional_emails": {
					Type:        framework.TypeStringSlice,
					Description: "additional_emails",
					Required:    true,
				},
				"mobile_phone": {
					Type:        framework.TypeString,
					Description: "mobile_phone",
					Required:    true,
				},
				"additional_phones": {
					Type:        framework.TypeStringSlice,
					Description: "additional_phones",
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
		// Restore
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/user/" + uuid.Pattern("uuid") + "/restore" + "$",
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
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleRestore(),
					Summary:  "Restore the user by ID.",
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
					Description: "ID of the tenant user",
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
					Description: "ID of the tenant user",
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
					Description: "ID of the tenant user",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived user multipass",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
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

		obj, err := usecase.Users(tx, tenantID).GetByID(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.TenantUUID == tenantID
		return exists, nil
	}
}

func (b *userBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create user", "path", req.Path)
		id := getCreationID(expectID, data)
		tenantID := data.Get(model.TenantForeignPK).(string)
		user := &model.User{
			UUID:             id,
			TenantUUID:       tenantID,
			Identifier:       data.Get("identifier").(string),
			FirstName:        data.Get("first_name").(string),
			LastName:         data.Get("last_name").(string),
			DisplayName:      data.Get("display_name").(string),
			Email:            data.Get("email").(string),
			AdditionalEmails: data.Get("additional_emails").([]string),
			MobilePhone:      data.Get("mobile_phone").(string),
			AdditionalPhones: data.Get("additional_phones").([]string),
			Origin:           model.OriginIAM,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err := usecase.Users(tx, tenantID).Create(user); err != nil {
			msg := "cannot create user"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"user": user}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *userBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("update user", "path", req.Path)
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		user := &model.User{
			UUID:             id,
			TenantUUID:       tenantID,
			Version:          data.Get("resource_version").(string),
			Identifier:       data.Get("identifier").(string),
			FirstName:        data.Get("first_name").(string),
			LastName:         data.Get("last_name").(string),
			DisplayName:      data.Get("display_name").(string),
			Email:            data.Get("email").(string),
			AdditionalEmails: data.Get("additional_emails").([]string),
			MobilePhone:      data.Get("mobile_phone").(string),
			AdditionalPhones: data.Get("additional_phones").([]string),
			Origin:           model.OriginIAM,
		}

		err := usecase.Users(tx, tenantID).Update(user)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"user": user}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *userBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete user", "path", req.Path)
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.Users(tx, tenantID).Delete(model.OriginIAM, id)
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
		b.Logger().Debug("read user", "path", req.Path)
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)

		user, err := usecase.Users(tx, tenantID).GetByID(id)
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"user": user}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *userBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list users", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)

		users, err := usecase.Users(tx, tenantID).List(showArchived)
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"users": users,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *userBackend) handleRestore() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("restore user", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)

		user, err := usecase.Users(tx, tenantID).Restore(id)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"user": user,
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *userBackend) handleMultipassCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create user multipass", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		// Check that the feature is available
		if err := isJwtEnabled(tx, b.tokenController); err != nil {
			return responseErr(req, err)
		}

		var (
			tid = data.Get("tenant_uuid").(string)
			uid = data.Get("owner_uuid").(string)

			ttl         = time.Duration(data.Get("ttl").(int)) * time.Second
			maxTTL      = time.Duration(data.Get("max_ttl").(int)) * time.Second
			cidrs       = data.Get("allowed_cidrs").([]string)
			roles       = data.Get("allowed_roles").([]string)
			description = data.Get("description").(string)
		)

		issueFn := jwt.CreateIssueMultipassFunc(b.tokenController, tx)
		jwtString, multipass, err := usecase.
			UserMultipasses(tx, model.OriginIAM, tid, uid).
			CreateWithJWT(issueFn, ttl, maxTTL, cidrs, roles, description)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"multipass": multipass,
			"token":     jwtString,
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *userBackend) handleMultipassDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete user multipass", "path", req.Path)
		var (
			id  = data.Get("uuid").(string)
			tid = data.Get("tenant_uuid").(string)
			uid = data.Get("owner_uuid").(string)
		)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.UserMultipasses(tx, model.OriginIAM, tid, uid).Delete(id)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}
		return logical.RespondWithStatusCode(nil, nil, http.StatusNoContent)
	}
}

func (b *userBackend) handleMultipassRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read user multipass", "path", req.Path)
		var (
			id  = data.Get("uuid").(string)
			tid = data.Get("tenant_uuid").(string)
			uid = data.Get("owner_uuid").(string)
		)
		tx := b.storage.Txn(false)

		mp, err := usecase.UserMultipasses(tx, model.OriginIAM, tid, uid).GetByID(id)
		if err != nil {
			return responseErr(req, err)
		}
		resp := &logical.Response{Data: map[string]interface{}{"multipass": model.OmitSensitive(mp)}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *userBackend) handleMultipassList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list user multipasses", "path", req.Path)
		tid := data.Get("tenant_uuid").(string)
		uid := data.Get("owner_uuid").(string)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}

		tx := b.storage.Txn(false)

		multipasses, err := usecase.UserMultipasses(tx, model.OriginIAM, tid, uid).PublicList(showArchived)
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"multipasses": multipasses,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
