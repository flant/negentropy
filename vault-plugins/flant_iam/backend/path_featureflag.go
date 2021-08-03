package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type featureFlagBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func featureFlagPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &featureFlagBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b featureFlagBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Create, update
		{
			Pattern: "feature_flag",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeNameString,
					Description: "Feature flag name",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create feature flag.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create feature flag.",
				},
			},
		},
		{
			Pattern: "feature_flag/?",
			Fields: map[string]*framework.FieldSchema{
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived feature flags",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all feature flags.",
				},
			},
		},
		// Read, update, delete by name
		{
			Pattern: "feature_flag/" + framework.GenericNameRegex("name") + "$",
			Fields: map[string]*framework.FieldSchema{

				"name": {
					Type:        framework.TypeNameString,
					Description: "Feature flag name",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the featureFlag by ID.",
				},
			},
		},
	}
}

func (b *featureFlagBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		name := data.Get("name").(string)
		b.Logger().Debug("checking featureFlag existence", "path", req.Path, "name", name, "op", req.Operation)

		tx := b.storage.Txn(false)
		repo := model.NewFeatureFlagRepository(tx)

		flag, err := repo.GetByID(name)
		if err != nil {
			return false, err
		}
		return flag != nil, nil
	}
}

func (b *featureFlagBackend) handleCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create feature_flag", "path", req.Path)
		featureFlag := &model.FeatureFlag{
			Name: data.Get("name").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err := usecase.Featureflags(tx).Create(featureFlag); err != nil {
			msg := "cannot create feature flag"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"feature_flag": featureFlag}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *featureFlagBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete feature_flag", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		name := data.Get("name").(string)

		err := usecase.Featureflags(tx).Delete(name)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *featureFlagBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list feature_flags", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}
		tx := b.storage.Txn(false)

		list, err := usecase.Featureflags(tx).List(showArchived)
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
