package backend

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func (b *tenantBackend) featureFlagPath() *framework.Path {
	// Feature flags for tenant
	return &framework.Path{
		Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/feature_flag/" + framework.GenericNameRegex("feature_flag_name"),
		Fields: map[string]*framework.FieldSchema{
			"tenant_uuid": {
				Type:        framework.TypeNameString,
				Description: "ID of a tenant",
				Required:    true,
			},
			"feature_flag_name": {
				Type:        framework.TypeNameString,
				Description: "Feature flag's name",
				Required:    true,
			},
			"enabled_for_new_projects": {
				Type:        framework.TypeBool,
				Description: "Enable by default for a new projects inside this tenant",
				Default:     true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagBinding(),
				Summary:  "Add Feature flag to the tenant.",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagBinding(),
				Summary:  "Add Feature flag to the tenant.",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.handleFeatureFlagDelete(),
				Summary:  "Remove Feature flag from the tenant.",
			},
		},
	}
}

func (b *tenantBackend) handleFeatureFlagBinding() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)
		featureFlagName := data.Get("feature_flag_name").(string)
		enabledByDefault := data.Get("enabled_for_new_projects").(bool)

		if featureFlagName == "" {
			return nil, logical.CodedError(http.StatusBadRequest, "feature_flag_name required")
		}

		tff := model.TenantFeatureFlag{
			FeatureFlag:           model.FeatureFlag{Name: featureFlagName},
			EnabledForNewProjects: enabledByDefault,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		tenant, err := usecase.TenantFeatureFlags(tx, tenantID).Add(tff)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"tenant": tenant}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *tenantBackend) handleFeatureFlagDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)
		featureFlagName := data.Get("feature_flag_name").(string)

		if featureFlagName == "" {
			return nil, logical.CodedError(http.StatusBadRequest, "feature_flag_name required")
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		tenant, err := usecase.TenantFeatureFlags(tx, tenantID).Delete(featureFlagName)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"tenant": tenant}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
