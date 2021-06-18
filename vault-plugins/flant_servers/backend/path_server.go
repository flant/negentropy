package backend

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	servermodel "github.com/flant/negentropy/vault-plugins/flant_servers/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// TODO: changed group identifier once project indentifier changes in the flant_iam

type serverBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func serverPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &serverBackend{
		Backend: b,
		storage: storage,
	}

	return bb.paths()
}

func (b *serverBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: path.Join("tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "register_server"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"labels": {
					Type:        framework.TypeMap,
					Description: "Map of labels",
					Required:    false,
				},
				"annotations": {
					Type:        framework.TypeMap,
					Description: "Map of annotations",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleRegister(),
					Summary:  "Register server",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleRegister(),
					Summary:  "Register server",
				},
			},
		},
		{
			Pattern: path.Join(
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "server", uuid.Pattern("server_uuid")),
			Fields: map[string]*framework.FieldSchema{
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
				},
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
				"server_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a server",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
				},
				"labels": {
					Type:        framework.TypeMap,
					Description: "Map of labels",
				},
				"annotations": {
					Type:        framework.TypeMap,
					Description: "Map of annotations",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Read a server",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update a server",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Delete a server",
				},
			},
		},
		// TODO: label selector for lists?
		{
			Pattern: path.Join(
				"tenant", uuid.Pattern("tenant_uuid"), "project", uuid.Pattern("project_uuid"), "servers.*"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"project_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "List servers",
				},
			},
		},
	}
}

func (b *serverBackend) handleRegister() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		// TODO: clarify use of getCreationID()
		id := uuid.New()

		var (
			labels      = make(map[string]string)
			annotations = make(map[string]string)
		)
		for k, v := range data.Get("labels").(map[string]interface{}) {
			labels[k] = v.(string)
		}
		for k, v := range data.Get("annotations").(map[string]interface{}) {
			annotations[k] = v.(string)
		}

		server := &servermodel.Server{
			UUID:        id,
			TenantUUID:  data.Get(model.TenantForeignPK).(string),
			ProjectUUID: data.Get(model.ProjectForeignPK).(string),
			Identifier:  data.Get("identifier").(string),
			Labels:      labels,
			Annotations: annotations,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewServerRepository(tx)

		if err := repo.Create(server); err != nil {
			msg := "cannot create server"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), err
		}

		err := tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), err
		}

		// TODO: token creation

		return responseWithDataAndCode(req, server, http.StatusCreated)
	}
}

func (b *serverBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		defer tx.Abort()
		repo := NewServerRepository(tx)

		server, err := repo.GetById(data.Get("server_uuid").(string))
		if err != nil {
			err := fmt.Errorf("cannot get server from db: %s", err)
			b.Logger().Debug("err", err.Error())
			return logical.ErrorResponse(err.Error()), err
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		return responseWithDataAndCode(req, server, http.StatusCreated)
	}
}

func (b *serverBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewServerRepository(tx)

		var (
			labels      = make(map[string]string)
			annotations = make(map[string]string)
		)
		for k, v := range data.Get("labels").(map[string]interface{}) {
			labels[k] = v.(string)
		}
		for k, v := range data.Get("annotations").(map[string]interface{}) {
			annotations[k] = v.(string)
		}

		server := &servermodel.Server{
			UUID:        data.Get("server_uuid").(string),
			TenantUUID:  data.Get(model.TenantForeignPK).(string),
			ProjectUUID: data.Get(model.ProjectForeignPK).(string),
			Version:     data.Get("resource_version").(string),
			Identifier:  data.Get("identifier").(string),
			Labels:      labels,
			Annotations: annotations,
		}

		err := repo.Update(server)
		if errors.Is(err, ErrNotFound) {
			return responseNotFound(req, servermodel.ServerType)
		}
		if errors.Is(err, ErrVersionMismatch) {
			return responseVersionMismatch(req)
		}
		if err != nil {
			b.Logger().Debug("err", err.Error())
			return logical.ErrorResponse(err.Error()), nil
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		return responseWithDataAndCode(req, server, http.StatusCreated)
	}
}

func (b *serverBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("server_uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewServerRepository(tx)

		err := repo.Delete(id)
		if err == ErrNotFound {
			return responseNotFound(req, servermodel.ServerType)
		}
		if err != nil {
			return nil, err
		}

		err = tx.Commit()
		if err != nil {
			msg := "cannot commit transaction"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *serverBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)
		projectID := data.Get(model.ProjectForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := NewServerRepository(tx)

		list, err := repo.List(tenantID, projectID)
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

type ServerRepository struct {
	db *io.MemoryStoreTxn
}

func NewServerRepository(tx *io.MemoryStoreTxn) *ServerRepository {
	return &ServerRepository{
		db: tx,
	}
}

func (r *ServerRepository) Create(server *servermodel.Server) error {
	var (
		tenant         *model.Tenant
		project        *model.Project
		group          *model.Group
		roleBinding    *model.RoleBinding
		serviceAccount *model.ServiceAccount
	)

	rawTenant, err := r.db.First(model.TenantType, model.PK, server.TenantUUID)
	if err != nil {
		return err
	}
	if rawTenant == nil {
		return ErrNotFound
	}
	tenant = rawTenant.(*model.Tenant)

	rawProject, err := r.db.First(model.ProjectType, model.PK, server.ProjectUUID)
	if err != nil {
		return err
	}
	if rawProject == nil {
		return ErrNotFound
	}
	project = rawProject.(*model.Project)

	rawServer, err := r.db.First(servermodel.ServerType, "identifier")
	if err != nil {
		return err
	}
	if rawServer != nil {
		return fmt.Errorf("server with identifier %q already exists")
	}

	// TODO: <builtin_type>
	rawGroup, err := r.db.First(model.GroupType, "full_identifier", fmt.Sprintf("servers@%s.<builtin_type>.groups.%s",
		project.Identifier, tenant.Identifier))
	if err != nil {
		return err
	}
	if rawGroup != nil {
		group = rawGroup.(*model.Group)
	}

	// TODO: everything roleBinding
	rawRoleBinding, err := r.db.First(model.RoleBindingType, "full_identifier", fmt.Sprintf("%s@<builtin_group_type>.groups.%s",
		server.Identifier, tenant.Identifier))
	if err != nil {
		return err
	}
	if rawRoleBinding != nil {
		roleBinding = rawRoleBinding.(*model.RoleBinding)
	}

	// TODO: <builtin_type>
	rawServiceAccount, err := r.db.First(model.ServiceAccountType, "full_identifier", fmt.Sprintf("%s@%s.<builtin_type>.groups.%s",
		server.Identifier, project.Identifier, tenant.Identifier))
	if err != nil {
		return err
	}
	if rawServiceAccount != nil {
		serviceAccount = rawServiceAccount.(*model.ServiceAccount)
	}

	if serviceAccount == nil {
		newServiceAccount := &model.ServiceAccount{
			UUID:        uuid.New(),
			TenantUUID:  tenant.ObjId(),
			Version:     model.NewResourceVersion(),
			BuiltinType: "", // TODO
			Identifier:  server.Identifier + "@" + project.Identifier,
		}
		newServiceAccount.FullIdentifier = model.CalcServiceAccountFullIdentifier(newServiceAccount, tenant)

		err := r.db.Insert(model.ServiceAccountType, newServiceAccount)
		if err != nil {
			return err
		}

		serviceAccount = newServiceAccount
	}

	if group == nil {
		newGroup := &model.Group{
			UUID:            uuid.New(),
			TenantUUID:      tenant.ObjId(),
			Version:         model.NewResourceVersion(),
			BuiltinType:     "", // TODO
			Identifier:      "servers@" + project.Identifier,
			ServiceAccounts: []string{serviceAccount.ObjId()},
		}
		newGroup.FullIdentifier = model.CalcGroupFullIdentifier(newGroup, tenant)

		err := r.db.Insert(model.GroupType, newGroup)
		if err != nil {
			return err
		}

		group = newGroup
	}

	if roleBinding == nil {
		newRoleBinding := &model.RoleBinding{
			UUID:            uuid.New(),
			TenantUUID:      server.TenantUUID,
			Version:         model.NewResourceVersion(),
			BuiltinType:     "", // TODO
			ServiceAccounts: []string{serviceAccount.ObjId()},
			Roles:           []model.BoundRole{}, // TODO
		}
		newRoleBinding.FullIdentifier = model.CalcRoleBindingFullIdentifier(newRoleBinding, tenant)

		err := r.db.Insert(model.RoleBindingType, newRoleBinding)
		if err != nil {
			return err
		}

		roleBinding = newRoleBinding
	}

	server.Version = model.NewResourceVersion()

	err = r.db.Insert(servermodel.ServerType, server)
	if err != nil {
		return err
	}

	return nil
}

func (r *ServerRepository) GetById(id string) (*servermodel.Server, error) {
	raw, err := r.db.First(servermodel.ServerType, model.PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, backend.ErrNotFound
	}

	server := raw.(*servermodel.Server)
	return server, nil
}

func (r *ServerRepository) Update(server *servermodel.Server) error {
	stored, err := r.GetById(server.UUID)
	if err != nil {
		return err
	}

	if stored.TenantUUID != server.TenantUUID {
		return backend.ErrNotFound
	}
	if stored.Version != server.Version {
		return backend.ErrVersionMismatch
	}
	server.Version = model.NewResourceVersion()

	rawTenant, err := r.db.First(model.TenantType, model.PK, server.TenantUUID)
	if err != nil {
		return err
	}
	if rawTenant == nil {
		return backend.ErrNotFound
	}
	tenant := rawTenant.(*model.Tenant)

	server.FullIdentifier = server.Identifier + "@" + tenant.Identifier

	err = r.db.Insert(servermodel.ServerType, server)
	if err != nil {
		return err
	}

	return nil
}

func (r *ServerRepository) Delete(id string) error {
	server, err := r.GetById(id)
	if err != nil {
		return err
	}

	return r.db.Delete(servermodel.ServerType, server)
}

func (r *ServerRepository) List(tenantID, projectID string) ([]string, error) {
	iter, err := r.db.Get(servermodel.ServerType, "tenant_project", tenantID, projectID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*servermodel.Server)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}
