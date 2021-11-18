package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type roleBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func rolePaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &roleBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b roleBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "role",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeNameString,
					Description: "Role name",
					Required:    true,
				},
				"description": {
					Type:        framework.TypeString,
					Description: "Role description",
					Required:    true,
				},
				"scope": {
					Type:        framework.TypeString,
					Description: "The scope of the role",
					Required:    true,
					AllowedValues: []interface{}{
						model.RoleScopeProject,
						model.RoleScopeProject,
					},
				},
				"options_schema": {
					Type:        framework.TypeString,
					Description: "JSON schema of the role options",
					Required:    true,
				},
				"require_one_of_feature_flags": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Enumerated flags, one of which is required in the scope to use role",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create role",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create role",
				},
			},
		},
		// List
		{
			Pattern: "role/?",
			Fields: map[string]*framework.FieldSchema{
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived roles",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all roles IDs",
				},
			},
		},
		// Read, update, delete by name
		{

			Pattern: "role/" + framework.GenericNameRegex("name") + "$",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeNameString,
					Description: "Role name, unique globally",
					Required:    true,
				},
				"description": {
					Type:        framework.TypeString,
					Description: "Role description",
					Required:    true,
				},
				// changing type is forbidden
				"require_one_of_feature_flags": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Enumerated flags, one of which is required in the scope to use role",
					Required:    true,
				},
				"options_schema": {
					Type:        framework.TypeString,
					Description: "JSON schema of the role options",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the role by ID",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the role by ID",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the role by ID",
				},
			},
		},
		// Include/exclude inherited role
		{

			Pattern: "role/" + framework.GenericNameRegex("name") + "/include/" + framework.GenericNameRegex("included_name") + "$",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeNameString,
					Description: "Destination role name",
					Required:    true,
				},
				"included_name": {
					Type:        framework.TypeNameString,
					Description: "Role name to include",
					Required:    true,
				},
				"options_template": {
					Type:        framework.TypeString,
					Description: "Go template to use outermost values in the included role schema",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleInclude(),
					Summary:  "Include role",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleExclude(),
					Summary:  "Exclude role",
				},
			},
		},
	}
}

func (b *roleBackend) handleCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("creating role", "path", req.Path)

		roleType := data.Get("scope").(string)
		role := &model.Role{
			Name:                     data.Get("name").(string),
			Scope:                    model.RoleScope(roleType),
			Description:              data.Get("description").(string),
			OptionsSchema:            data.Get("options_schema").(string),
			RequireOneOfFeatureFlags: data.Get("require_one_of_feature_flags").([]string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err := usecase.Roles(tx).Create(role); err != nil {
			msg := "cannot create role"
			b.Logger().Debug(msg, "err", err.Error())
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"role": role}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *roleBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("updating role", "path", req.Path)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		role := &model.Role{
			Name:                     data.Get("name").(string),
			Description:              data.Get("description").(string),
			OptionsSchema:            data.Get("options_schema").(string),
			RequireOneOfFeatureFlags: data.Get("require_one_of_feature_flags").([]string),
		}

		err := usecase.Roles(tx).Update(role)
		if err != nil {
			return responseErr(req, err)
		}
		if err = commit(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"role": role}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("deleting role", "path", req.Path)

		name := data.Get("name").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.Roles(tx).Delete(name)
		if err != nil {
			return responseErr(req, err)
		}
		if err = commit(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *roleBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("getting role", "path", req.Path)

		name := data.Get("name").(string)

		tx := b.storage.Txn(false)

		role, err := usecase.Roles(tx).Get(name)
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"role": role}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("listing roles", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}

		tx := b.storage.Txn(false)
		repo := iam_repo.NewRoleRepository(tx)

		list, err := repo.List(showArchived)
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"names": list,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBackend) handleInclude() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("including role", "path", req.Path)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		var (
			destName = data.Get("name").(string)
			srcName  = data.Get("included_name").(string)
			template = data.Get("options_template").(string)
		)

		incl := &model.IncludedRole{
			Name:            srcName,
			OptionsTemplate: template,
		}

		err := usecase.Roles(tx).Include(destName, incl)
		if err != nil {
			return responseErr(req, err)
		}
		if err = commit(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *roleBackend) handleExclude() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("excluding role", "path", req.Path)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		var (
			destName = data.Get("name").(string)
			srcName  = data.Get("included_name").(string)
		)

		err := usecase.Roles(tx).Exclude(destName, srcName)
		if err != nil {
			return responseErr(req, err)
		}
		if err = commit(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}
