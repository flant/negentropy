package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
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
		// Creation
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
		// List
		{
			Pattern: "feature_flag/?",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
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

		flag, err := repo.Get(name)
		if err != nil {
			return false, err
		}
		return flag != nil, nil
	}
}

func (b *featureFlagBackend) handleCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		featureFlag := &model.FeatureFlag{
			Name: data.Get("name").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewFeatureFlagRepository(tx)

		if err := repo.Create(featureFlag); err != nil {
			msg := "cannot create feature flag"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, featureFlag, http.StatusCreated)
	}
}

func (b *featureFlagBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewFeatureFlagRepository(tx)

		name := data.Get("name").(string)
		err := repo.Delete(name)
		if err == model.ErrNotFound {
			return responseNotFound(req)
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

func (b *featureFlagBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)
		repo := model.NewFeatureFlagRepository(tx)

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
