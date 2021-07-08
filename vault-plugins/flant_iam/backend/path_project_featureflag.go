package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func (b *projectBackend) featureFlagPath() *framework.Path {
	return &framework.Path{
		Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/project/" + uuid.Pattern("uuid") + "/feature_flag/" + framework.GenericNameRegex("feature_flag_name"),
		Fields: map[string]*framework.FieldSchema{
			"tenant_uuid": {
				Type:        framework.TypeNameString,
				Description: "ID of a tenant",
				Required:    true,
			},
			"uuid": {
				Type:        framework.TypeNameString,
				Description: "ID of a project",
				Required:    true,
			},
			"feature_flag_name": {
				Type:        framework.TypeNameString,
				Description: "Feature flag's name",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagBinding(),
				Summary:  "Add FeatureFlag to project.",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagBinding(),
				Summary:  "Add FeatureFlag to project.",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagDelete(),
				Summary:  "Remove FeatureFlag from project.",
			},
		},
	}
}

func (b *projectBackend) handleFeatureFlagBinding() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)
		projectID := data.Get("uuid").(string)
		featureFlagName := data.Get("feature_flag_name").(string)
		if featureFlagName == "" {
			return nil, logical.CodedError(http.StatusBadRequest, "feature_flag_name required")
		}

		tff := model.FeatureFlag{
			Name: featureFlagName,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		repo := model.NewProjectFeatureFlagRepository(tx)

		project, err := repo.SetFlagToProject(tenantID, projectID, tff)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, project, http.StatusOK)
	}
}

func (b *projectBackend) handleFeatureFlagDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)
		projectID := data.Get("uuid").(string)
		featureFlagName := data.Get("feature_flag_name").(string)

		if featureFlagName == "" {
			return nil, logical.CodedError(http.StatusBadRequest, "feature_flag_name required")
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		repo := model.NewProjectFeatureFlagRepository(tx)

		project, err := repo.RemoveFlagFromProject(tenantID, projectID, featureFlagName)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, project, http.StatusOK)
	}
}
