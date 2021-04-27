package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

type userBackend struct {
	logical.Backend
	storage *memdb.MemDB
}

func userPaths(b logical.Backend, storage *memdb.MemDB) []*framework.Path {
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
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the user by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the user by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the user by ID.",
				},
			},
		},
	}
}

func (b *userBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		b.Logger().Debug("existance", "path", req.Path, "data", data)

		id := data.Get("uuid").(string)
		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}
		tx := b.storage.Txn(false)

		raw, err := tx.First(model.UserType, model.ID, id)
		if err != nil {
			return false, err
		}

		return raw != nil, nil
	}
}

func (b *userBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		var id string

		if expectID {
			// for privileged access
			id = data.Get("uuid").(string)
		}

		if id == "" {
			id = uuid.New()
		}

		user := &model.User{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
		}

		// Validation

		// TODO: validation should depend on the storage
		//      validate field uniqueness
		//      validate resource_version
		// feature flags

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := tx.Insert(model.UserType, user)
		if err != nil {
			b.Logger().Debug("cannot create user", "err", err.Error())
			return logical.ErrorResponse("cannot create user"), nil
		}
		defer tx.Commit()

		// Response

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuid": id,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *userBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		raw, err := tx.First(model.UserType, model.ID, id)
		if err != nil {
			return nil, err
		}
		if raw == nil {
			rr := logical.ErrorResponse("user not found")
			return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
		}

		user := raw.(*model.User)
		// user.TenantUUID = data.Get(model.TenantForeignPK).(string)

		// Validation

		// TODO: validation should depend on the storage
		//      validate field uniqueness
		//      validate resource_version
		// feature flags

		err = tx.Insert(model.UserType, user)
		if err != nil {
			b.Logger().Debug("cannot save user", "err", err.Error())
			return logical.ErrorResponse("cannot save user"), nil
		}
		defer tx.Commit()

		// Response

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuid": id,
			},
		}

		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *userBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(true)
		defer tx.Abort()

		// Verify existence

		id := data.Get("uuid").(string)
		raw, err := tx.First(model.UserType, model.ID, id)
		if err != nil {
			return nil, err
		}
		if raw == nil {
			rr := logical.ErrorResponse("user not found")
			return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
		}

		// Delete

		// FIXME: cascade deletion, e.g. deleteUser()
		err = tx.Delete(model.UserType, raw)
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		// Respond

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *userBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		id := data.Get("uuid").(string)

		// Find

		raw, err := tx.First(model.UserType, model.ID, id)
		if err != nil {
			return nil, err
		}
		if raw == nil {
			rr := logical.ErrorResponse("user not found")
			return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
		}

		// Respond

		user := raw.(*model.User)
		userJSON, err := user.Marshal(false)
		if err != nil {
			return nil, err
		}

		var responseData map[string]interface{}
		err = jsonutil.DecodeJSON(userJSON, &responseData)
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: responseData,
		}

		return resp, nil
	}
}

// nolint:unused
func (b *userBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		tid := data.Get("tenant_uuid").(string)

		// Find

		iter, err := tx.Get(model.UserType, model.TenantForeignPK, tid)
		if err != nil {
			return nil, err
		}

		users := []string{}
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			t := raw.(*model.User)
			users = append(users, t.UUID)
		}

		// Respond

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuids": users,
			},
		}

		return resp, nil
	}
}
