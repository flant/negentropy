package jwtauth

import (
	"context"
	"net/http"
	"path"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extension_server_access/model"
)

func pathTenant(b *flantIamAuthBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "tenant/?",
			Fields:  map[string]*framework.FieldSchema{},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.listTenants,
					Summary:  "List all tenants.",
				},
			},
		},
		{
			Pattern: "tenant/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.readTenant,
					Summary:  "Retrieve the tenant by ID.",
				},
			},
		},
		{
			Pattern: path.Join("tenant", uuid.Pattern("tenant_uuid"), "project/?"),
			Fields: map[string]*framework.FieldSchema{
				"tenant_uuid": {
					Type:        framework.TypeString,
					Description: "UUID of a tenant",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.listProjects,
					Summary:  "List all projects of a tenant.",
				},
			},
		},
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
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.readProject,
					Summary:  "Retrieve the project by ID.",
				},
			},
		},
	}
}

// UNSAFE : only uuids and versions in response
func (b *flantIamAuthBackend) listTenants(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	// TODO remove not user tenants
	txn := b.storage.Txn(false)
	defer txn.Abort()

	b.Logger().Debug("got tenant request in auth")

	repo := iam_repo.NewTenantRepository(txn)

	tenants, err := repo.List(false)
	if err != nil {
		return nil, err
	}

	b.Logger().Debug("list", "tenants", tenants)

	result := make([]ext.SafeTenant, 0, len(tenants))

	for _, tenant := range tenants {
		res := ext.SafeTenant{
			UUID:    tenant.UUID,
			Version: tenant.Version,
		}
		result = append(result, res)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"tenants": result},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *flantIamAuthBackend) listProjects(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// TODO remove not user projects
	tenantID := data.Get("tenant_uuid").(string)

	txn := b.storage.Txn(false)
	defer txn.Abort()

	repo := iam_repo.NewProjectRepository(txn)

	projects, err := repo.List(tenantID, false)
	if err != nil {
		return nil, err
	}

	result := make([]ext.SafeProject, 0, len(projects))

	for _, project := range projects {
		res := ext.SafeProject{
			UUID:       project.UUID,
			TenantUUID: project.TenantUUID,
			Version:    project.Version,
		}
		result = append(result, res)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"projects": result},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *flantIamAuthBackend) readTenant(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("read tenant", "path", req.Path)
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(false)

	tenant, err := usecase.Tenants(tx).GetByID(id)
	if err != nil {
		return responseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"tenant": tenant,
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *flantIamAuthBackend) readProject(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("read project", "path", req.Path)

	id := data.Get("uuid").(string)

	tx := b.storage.Txn(false)

	project, err := usecase.Projects(tx).GetByID(id)
	if err != nil {
		return responseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"project": &ext.Project{
			UUID:       project.UUID,
			TenantUUID: project.TenantUUID,
			Version:    project.Version,
			Identifier: project.Identifier,
		},
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
