package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

type projectBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func projectPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &projectBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b projectBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/project",
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
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/project/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
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
		// Listing
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/project/?",
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
					Summary:  "Lists all projects IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{

			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/project/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a project",
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
	}
}

func (b *projectBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		tenantID := data.Get(model.TenantForeignPK).(string)
		b.Logger().Debug("checking project existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		repo := NewProjectRepository(tx)

		obj, err := repo.GetById(id)
		if err != nil {
			return false, err
		}
		exists := obj != nil && obj.TenantUUID == tenantID
		return exists, nil
	}
}

func (b *projectBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := getCreationID(expectID, data)
		project := &model.Project{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Identifier: data.Get("identifier").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewProjectRepository(tx)

		if err := repo.Create(project); err != nil {
			msg := "cannot create project"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, project, http.StatusCreated)
	}
}

func (b *projectBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		project := &model.Project{
			UUID:       id,
			TenantUUID: data.Get(model.TenantForeignPK).(string),
			Version:    data.Get("resource_version").(string),
			Identifier: data.Get("identifier").(string),
		}

		repo := NewProjectRepository(tx)
		err := repo.Update(project)
		if err == ErrNotFound {
			return responseNotFound(req, model.ProjectType)
		}
		if err == ErrVersionMismatch {
			return responseVersionMismatch(req)
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return responseWithDataAndCode(req, project, http.StatusOK)
	}
}

func (b *projectBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := NewProjectRepository(tx)

		err := repo.Delete(id)
		if err == ErrNotFound {
			return responseNotFound(req, "project not found")
		}
		if err != nil {
			return nil, err
		}
		defer tx.Commit()

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *projectBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := NewProjectRepository(tx)

		project, err := repo.GetById(id)
		if err == ErrNotFound {
			return responseNotFound(req, model.ProjectType)
		}
		if err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, project, http.StatusOK)
	}
}

func (b *projectBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := NewProjectRepository(tx)

		list, err := repo.List(tenantID)
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

type ProjectRepository struct {
	db         *memdb.Txn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewProjectRepository(tx *memdb.Txn) *ProjectRepository {
	return &ProjectRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r ProjectRepository) Create(project *model.Project) error {
	_, err := r.tenantRepo.GetById(project.TenantUUID)
	if err != nil {
		return err
	}

	project.Version = model.NewResourceVersion()
	err = r.db.Insert(model.ProjectType, project)
	if err != nil {
		return err
	}
	return nil
}

func (r ProjectRepository) GetById(id string) (*model.Project, error) {
	raw, err := r.db.First(model.ProjectType, model.ID, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	project := raw.(*model.Project)

	return project, nil
}

func (r ProjectRepository) Update(project *model.Project) error {
	stored, err := r.GetById(project.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != project.TenantUUID {
		return ErrNotFound
	}
	if stored.Version != project.Version {
		return ErrVersionMismatch
	}
	project.Version = model.NewResourceVersion()

	// Update

	err = r.db.Insert(model.ProjectType, project)
	if err != nil {
		return err
	}

	return nil
}

func (r ProjectRepository) Delete(id string) error {
	project, err := r.GetById(id)
	if err != nil {
		return err
	}

	return r.db.Delete(model.ProjectType, project)
}

func (r ProjectRepository) List(tenantID string) ([]string, error) {
	iter, err := r.db.Get(model.ProjectType, model.TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*model.Project)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}
