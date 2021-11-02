package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func (b *projectBackend) featureFlagPath() *framework.Path {
	return &framework.Path{
		Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/project/" + uuid.Pattern("project_uuid") + "/feature_flag/" + framework.GenericNameRegex("feature_flag_name"),
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
			"feature_flag_name": {
				Type:        framework.TypeNameString,
				Description: "Feature flag's name",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagBinding(),
				Summary:  "Add FeatureFlag to the project.",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagBinding(),
				Summary:  "Add FeatureFlag to the project.",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagDelete(),
				Summary:  "Remove FeatureFlag from the project.",
			},
		},
	}
}

func (b *projectBackend) handleFeatureFlagBinding() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)
		projectID := data.Get("project_uuid").(string)
		featureFlagName := data.Get("feature_flag_name").(string)
		if featureFlagName == "" {
			return nil, logical.CodedError(http.StatusBadRequest, "feature_flag_name required")
		}

		ff := model.FeatureFlag{
			Name: featureFlagName,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		project, err := usecase.ProjectFeatureFlags(tx, tenantID, projectID).Add(ff)
		if err != nil {
			return responseErr(req, err)
		}

		if err = commit(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"project": project}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *projectBackend) handleFeatureFlagDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)
		projectID := data.Get("project_uuid").(string)
		featureFlagName := data.Get("feature_flag_name").(string)

		if featureFlagName == "" {
			return nil, logical.CodedError(http.StatusBadRequest, "feature_flag_name required")
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		project, err := usecase.ProjectFeatureFlags(tx, tenantID, projectID).Delete(featureFlagName)
		if err != nil {
			return responseErr(req, err)
		}

		if err = commit(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"project": project}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
