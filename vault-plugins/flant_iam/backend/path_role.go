package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/io"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
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
				"type": {
					Type:        framework.TypeString,
					Description: "Type denoting the scope of the group",
					Required:    true,
					AllowedValues: []interface{}{
						model.GroupScopeProject,
						model.GroupScopeTenant,
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
					Summary:  "Create role.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create role.",
				},
			},
		},
		// List
		{
			Pattern: "role/?",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all roles IDs.",
				},
			},
		},
		// Read, update, delete by uuid
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
					Summary:  "Update the role by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the role by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the role by ID.",
				},
			},
		},
	}
}

func (b *roleBackend) handleCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("creating role", "path", req.Path)

		roleType := data.Get("type").(string)
		role := &model.Role{
			Name:                     data.Get("name").(string),
			Type:                     model.GroupScope(roleType),
			Description:              data.Get("description").(string),
			OptionsSchema:            data.Get("options_schema").(string),
			RequireOneOfFeatureFlags: data.Get("require_one_of_feature_flags").([]string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewRoleRepository(tx)

		if err := repo.Create(role); err != nil {
			msg := "cannot create role"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, role, http.StatusCreated)
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

		repo := NewRoleRepository(tx)
		err := repo.Update(role)
		if err == ErrNotFound {
			return responseNotFound(req, model.RoleType)
		}
		if err != nil {
			return nil, err
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, role, http.StatusOK)
	}
}

func (b *roleBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("deleting role", "path", req.Path)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewRoleRepository(tx)

		name := data.Get("name").(string)
		err := repo.Delete(name)
		if err == ErrNotFound {
			return responseNotFound(req, "role not found")
		}
		if err != nil {
			return nil, err
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *roleBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("getting role", "path", req.Path)

		name := data.Get("name").(string)

		tx := b.storage.Txn(false)
		repo := NewRoleRepository(tx)

		role, err := repo.Get(name)
		if err == ErrNotFound {
			return responseNotFound(req, model.RoleType)
		}
		if err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, role, http.StatusOK)
	}
}

func (b *roleBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("listing roles", "path", req.Path)

		tx := b.storage.Txn(false)
		repo := NewRoleRepository(tx)

		list, err := repo.List()
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"names": list,
			},
		}
		return resp, nil
	}
}

type RoleRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics

}

func NewRoleRepository(tx *io.MemoryStoreTxn) *RoleRepository {
	return &RoleRepository{
		db: tx,
	}
}

func (r *RoleRepository) Create(t *model.Role) error {
	return r.db.Insert(model.RoleType, t)
}

func (r *RoleRepository) Get(name string) (*model.Role, error) {
	raw, err := r.db.First(model.RoleType, model.PK, name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*model.Role), nil
}

func (r *RoleRepository) Update(updated *model.Role) error {
	stored, err := r.Get(updated.Name)
	if err != nil {
		return err
	}

	updated.Type = stored.Type // type cannot be changed

	// TODO validate feature flags: role must not become unaccessable in the scope where it is used
	// TODO forbid backwards-incompatible changes of the options schema

	return r.db.Insert(model.RoleType, updated)
}

func (r *RoleRepository) Delete(name string) error {
	role, err := r.Get(name)
	if err != nil {
		return err
	}

	// TODO before the deletion, check it is not used in
	//      * role_biondings,
	//      * approvals,
	//      * tokens,
	//      * service account passwords

	return r.db.Delete(model.RoleType, role)
}

func (r *RoleRepository) List() ([]string, error) {
	iter, err := r.db.Get(model.RoleType, model.PK)
	if err != nil {
		return nil, err
	}

	names := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.Role)
		names = append(names, t.Name)
	}
	return names, nil
}
