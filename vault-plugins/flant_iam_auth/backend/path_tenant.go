package backend

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
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
	txn := b.storage.Txn(false)
	defer txn.Abort()

	b.Logger().Debug("got tenant request in auth")

	acceptedTenants, err := b.entityIDResolver.AvailableTenantsByEntityID(req.EntityID, txn)
	if err != nil {
		return backentutils.ResponseErrMessage(req, fmt.Sprintf("collect acceptedTenants: %s", err.Error()), http.StatusInternalServerError)
	}

	tenants, err := usecase.ListAvailableSafeTenants(txn, acceptedTenants)
	if err != nil {
		return backentutils.ResponseErrMessage(req, fmt.Sprintf("collect tenants: %s", err.Error()), http.StatusInternalServerError)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"tenants": tenants},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

// UNSAFE : only uuids and versions in response
func (b *flantIamAuthBackend) listProjects(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tenantID := data.Get("tenant_uuid").(string)

	txn := b.storage.Txn(false)
	defer txn.Abort()

	acceptedProjects, err := b.entityIDResolver.AvailableProjectsByEntityID(req.EntityID, txn)
	if err != nil {
		return backentutils.ResponseErrMessage(req, fmt.Sprintf("collect acceptedTenants & acceptedProjects: %s", err.Error()),
			http.StatusInternalServerError)
	}

	projects, err := usecase.ListAvailableSafeProjects(txn, tenantID, acceptedProjects)
	if err != nil {
		return backentutils.ResponseErrMessage(req, fmt.Sprintf("collect projects: %s", err.Error()), http.StatusInternalServerError)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"projects": projects},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *flantIamAuthBackend) readTenant(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("read tenant", "path", req.Path)
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(false)

	tenant, err := iam_usecase.Tenants(tx).GetByID(id)
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

	project, err := iam_usecase.Projects(tx).GetByID(id)
	if err != nil {
		return responseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"project": &model.Project{
			UUID:       project.UUID,
			TenantUUID: project.TenantUUID,
			Version:    project.Version,
			Identifier: project.Identifier,
		},
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
