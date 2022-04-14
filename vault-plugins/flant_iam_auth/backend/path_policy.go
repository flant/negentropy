package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type policyBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func policiesPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &policyBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b policyBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "login_policy",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeNameString,
					Description: "Negentropy policy name",
					Required:    true,
				},
				"rego": {
					Type:        framework.TypeString,
					Description: "Rego policy",
					Required:    true,
				},
				"roles": {
					Type:        framework.TypeStringSlice,
					Description: "Negentropy roles, which processed by this negentropy policy",
					Required:    true,
				},
				"claim_schema": {
					Type:        framework.TypeString,
					Description: "Open api specification for rego policy claim",
					Required:    true,
				},
				"allowed_auth_methods": {
					Type:        framework.TypeStringSlice,
					Description: "Allowed auth methods",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create policy.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create policy.",
				},
			},
		},
		// List
		{
			Pattern: "login_policy/?",
			Fields: map[string]*framework.FieldSchema{
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived policies",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all policies names.",
				},
			},
		},
		// Read, update, delete by uuid
		{
			Pattern: "login_policy/" + framework.GenericNameRegex("name") + "$",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeNameString,
					Description: "Negentropy policy name",
					Required:    true,
				},
				"rego": {
					Type:        framework.TypeNameString,
					Description: "Rego policy",
					Required:    true,
				},
				"roles": {
					Type:        framework.TypeStringSlice,
					Description: "Negentropy roles, which processed by this negentropy policy",
					Required:    true,
				},
				"claim_schema": {
					Type:        framework.TypeString,
					Description: "Open api specification for rego policy claim",
					Required:    true,
				},
				"allowed_auth_methods": {
					Type:        framework.TypeStringSlice,
					Description: "Allowed auth methods",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the policy by name.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the policy by name.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the policy by name.",
				},
			},
		},
	}
}

func (b *policyBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		name := data.Get("name").(string)
		b.Logger().Debug("checking policy existence", "path", req.Path, "name", name, "op", req.Operation)

		tx := b.storage.Txn(false)

		t, err := usecase.Policies(tx).GetByID(name)
		if err != nil {
			return false, err
		}
		return t != nil, nil
	}
}

func (b *policyBackend) handleCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create policy", "path", req.Path)

		policy := &model.Policy{
			Name:               data.Get("name").(string),
			Rego:               data.Get("rego").(string),
			Roles:              data.Get("roles").([]string),
			ClaimSchema:        data.Get("claim_schema").(string),
			AllowedAuthMethods: data.Get("allowed_auth_methods").([]string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err := usecase.Policies(tx).Create(policy); err != nil {
			msg := "cannot create policy"
			b.Logger().Error(msg, "err", err.Error())
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}
		if err := io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"policy": policy}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *policyBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("update policy", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		policy := &model.Policy{
			Name:               data.Get("name").(string),
			Rego:               data.Get("rego").(string),
			Roles:              data.Get("roles").([]string),
			ClaimSchema:        data.Get("claim_schema").(string),
			AllowedAuthMethods: data.Get("allowed_auth_methods").([]string),
		}

		err := usecase.Policies(tx).Update(policy)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"policy": policy}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *policyBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete policy", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		id := data.Get("name").(string)

		err := usecase.Policies(tx).Delete(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *policyBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read policy", "path", req.Path)
		id := data.Get("name").(string)

		tx := b.storage.Txn(false)

		policy, err := usecase.Policies(tx).GetByID(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"policy": policy,
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *policyBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("listing policies", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}

		tx := b.storage.Txn(false)
		policies, err := usecase.Policies(tx).List(showArchived)
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"policies": policies,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
