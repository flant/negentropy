package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_flow/iam_client"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_flow/usecase"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type projectBackend struct {
	logical.Backend
	storage       *io.MemoryStore
	projectClient iam_client.Projects
}

func projectPaths(b logical.Backend, storage *io.MemoryStore, projectClient iam_client.Projects) []*framework.Path {
	bb := &projectBackend{
		Backend:       b,
		storage:       storage,
		projectClient: projectClient,
	}
	return bb.paths()
}

func (b projectBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "client/" + uuid.Pattern("client_uuid") + "/project",
			Fields: map[string]*framework.FieldSchema{
				"client_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for project",
					Required:    true,
				},
				"service_packs": {
					Type:        framework.TypeKVPairs,
					Description: "Service packs",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create project.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create project.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "client/" + uuid.Pattern("client_uuid") + "/project/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
				"client_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"service_packs": {
					Type:        framework.TypeKVPairs,
					Description: "Service packs",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create project with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create project with preexistent ID.",
				},
			},
		},
		// List
		{
			Pattern: "client/" + uuid.Pattern("client_uuid") + "/project/?",
			Fields: map[string]*framework.FieldSchema{
				"client_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived projects",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all projects IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{
			Pattern: "client/" + uuid.Pattern("client_uuid") + "/project/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
					Required:    true,
				},
				"client_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
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
				"service_packs": {
					Type:        framework.TypeKVPairs,
					Description: "Service packs",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the project by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the project by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the project by ID.",
				},
			},
		},
		// Restore
		{
			Pattern: "client/" + uuid.Pattern("client_uuid") + "/project/" + uuid.Pattern("uuid") + "/restore" + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a user",
					Required:    true,
				},
				"client_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleRestore(),
					Summary:  "Restore the project by ID.",
				},
			},
		},
	}
}

func (b *projectBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		clientID := data.Get("client_uuid").(string)
		b.Logger().Debug("checking project existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)

		obj, err := usecase.Projects(tx, b.projectClient).GetByID(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.TenantUUID == clientID
		return exists, nil
	}
}

func (b *projectBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create project", "path", req.Path)

		id := getCreationID(expectID, data)

		servicePacks, err := b.getServicePacks(data)
		if err != nil {
			return responseErr(req, err)
		}

		project := &model.Project{
			Project: iam_model.Project{
				UUID:       id,
				TenantUUID: data.Get("client_uuid").(string),
				Identifier: data.Get("identifier").(string),
			},
			ServicePacks: servicePacks,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err := usecase.Projects(tx, b.projectClient).Create(project); err != nil {
			msg := "cannot create project"
			b.Logger().Error(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"project": project}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *projectBackend) getServicePacks(data *framework.FieldData) (map[model.ServicePackName]string, error) {
	servicePacksRaw := data.Get("service_packs")

	fmt.Printf("servicepacks: %#v ", servicePacksRaw) // TODO REMOVE

	servicePacks, ok := servicePacksRaw.(map[model.ServicePackName]string)
	if !ok {
		err := fmt.Errorf("marshalling params: wrong type of param service_packs, cant cast to map[model.ServicePackName]string, passed value:%#v",
			servicePacksRaw)
		b.Logger().Error(err.Error())
		return nil, err
	}
	return servicePacks, nil
}

func (b *projectBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("update project", "path", req.Path)

		servicePacks, err := b.getServicePacks(data)
		if err != nil {
			return responseErr(req, err)
		}

		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		project := &model.Project{
			Project: iam_model.Project{
				UUID:       id,
				TenantUUID: data.Get("client_uuid").(string),
				Version:    data.Get("resource_version").(string),
				Identifier: data.Get("identifier").(string),
			},
			ServicePacks: servicePacks,
		}

		err = usecase.Projects(tx, b.projectClient).Update(project)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"project": project}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *projectBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete project", "path", req.Path)

		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.Projects(tx, b.projectClient).Delete(id)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *projectBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read project", "path", req.Path)

		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)

		project, err := usecase.Projects(tx, b.projectClient).GetByID(id)
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"project": project}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *projectBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list projects", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}
		clientID := data.Get("client_uuid").(string)

		tx := b.storage.Txn(false)

		projects, err := usecase.Projects(tx, b.projectClient).List(clientID, showArchived)
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"projects": projects,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *projectBackend) handleRestore() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("restore project", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		id := data.Get("uuid").(string)

		project, err := usecase.Projects(tx, b.projectClient).Restore(id)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"project": project,
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
