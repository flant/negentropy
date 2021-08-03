package jwtauth

import (
	"context"
	"net/http"
	"path"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
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
	}
}

type uuidIdentifier struct {
	UUID       string `json:"uuid"`
	Identifier string `json:"identifier"`
}

func (b *flantIamAuthBackend) listTenants(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	txn := b.storage.Txn(false)
	defer txn.Abort()

	b.Logger().Debug("got tenant request in auth")

	repo := model.NewTenantRepository(txn)

	tenants, err := repo.List(false)
	if err != nil {
		return nil, err
	}

	b.Logger().Debug("list", "tenants", tenants)

	result := make([]uuidIdentifier, 0, len(tenants))

	for _, tenant := range tenants {
		res := uuidIdentifier{
			Identifier: tenant.Identifier,
			UUID:       tenant.UUID,
		}
		result = append(result, res)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"tenants": result},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *flantIamAuthBackend) listProjects(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tenantID := data.Get("tenant_uuid").(string)

	txn := b.storage.Txn(false)
	defer txn.Abort()

	repo := model.NewProjectRepository(txn)

	projects, err := repo.List(tenantID, false)
	if err != nil {
		return nil, err
	}

	result := make([]uuidIdentifier, 0, len(projects))

	for _, project := range projects {
		res := uuidIdentifier{
			Identifier: project.Identifier,
			UUID:       project.UUID,
		}
		result = append(result, res)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"projects": result},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
