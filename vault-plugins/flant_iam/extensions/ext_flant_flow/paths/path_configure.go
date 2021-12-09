package paths

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type flantFlowConfigureBackend struct {
	*flantFlowExtension
}

func flantFlowConfigurePaths(e *flantFlowExtension) []*framework.Path {
	bb := &flantFlowConfigureBackend{
		flantFlowExtension: e,
	}

	return bb.paths()
}

func (b *flantFlowConfigureBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: path.Join("configure_extension", "flant_flow", "flant_tenant", uuid.Pattern("flant_tenant_uuid")),
			Fields: map[string]*framework.FieldSchema{
				"flant_tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant which is Flant",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleConfigFlantTenant,
					Summary:  "Set Flant uuid",
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleConfigFlantTenant,
					Summary:  "Set Flant uuid",
				},
			},
		},
		{
			Pattern: path.Join("configure_extension", "flant_flow", "specific_roles"),
			Fields: map[string]*framework.FieldSchema{
				"specific_roles": {
					Type: framework.TypeKVPairs,
					Description: fmt.Sprintf("Mapping some specific keys to iam.RoleName, mandatory keys:%v",
						config.MandatorySpecificRoles),
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleConfigSpecificRoles,
					Summary:  "Set specific iam.roles for flant_flow extension",
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleConfigSpecificRoles,
					Summary:  "Set specific iam.roles for flant_flow extension",
				},
			},
		},
		{
			Pattern: path.Join("configure_extension", "flant_flow", "specific_teams"),
			Fields: map[string]*framework.FieldSchema{
				"specific_teams": {
					Type: framework.TypeKVPairs,
					Description: fmt.Sprintf("Mapping some specific keys to flant_flow.TeamUUID, mandatory keys:%v",
						config.MandatorySpecificTeams),
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleConfigSpecificTeams),
					Summary:  "Set specific teams for flant_flow extension",
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleConfigSpecificTeams),
					Summary:  "Set specific teams for flant_flow extension",
				},
			},
		},
	}
}

func (b *flantFlowConfigureBackend) handleConfigFlantTenant(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("handleConfig started")
	defer b.Logger().Info("handleConfig exit")
	txn := b.storage.Txn(true)
	defer txn.Commit() //nolint:errcheck
	flantUUID := data.Get("flant_tenant_uuid").(string)
	cfg, err := usecase.Config(txn).SetFlantTenantUUID(ctx, req.Storage, flantUUID)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	b.setLiveConfig(cfg)
	b.Logger().Info("handleConfig normal finish")
	return logical.RespondWithStatusCode(nil, req, http.StatusOK)
}

func (b *flantFlowConfigureBackend) handleConfigSpecificRoles(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("handleConfig started")
	defer b.Logger().Info("handleConfig exit")
	txn := b.storage.Txn(true)
	defer txn.Commit() //nolint:errcheck
	rolesMap := data.Get("specific_roles").(map[string]string)
	if len(rolesMap) == 0 {
		return backentutils.ResponseErr(req,
			fmt.Errorf("%w: mandatory param 'specific_roles' not passed, or is empty", consts.ErrInvalidArg))
	}
	cfg, err := usecase.Config(txn).UpdateSpecificRoles(ctx, req.Storage, rolesMap)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	b.setLiveConfig(cfg)

	b.Logger().Info("handleConfig normal finish")
	return logical.RespondWithStatusCode(nil, req, http.StatusOK)
}

func (b *flantFlowConfigureBackend) handleConfigSpecificTeams(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("handleConfig started")
	defer b.Logger().Info("handleConfig exit")
	txn := b.storage.Txn(true)
	defer txn.Commit() //nolint:errcheck
	teamsMap := data.Get("specific_teams").(map[string]string)
	if len(teamsMap) == 0 {
		return backentutils.ResponseErr(req,
			fmt.Errorf("%w: mandatory param 'specific_teams' not passed, or is empty", consts.ErrInvalidArg))
	}
	cfg, err := usecase.Config(txn).UpdateSpecificTeams(ctx, req.Storage, teamsMap)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	b.setLiveConfig(cfg)
	b.Logger().Info("handleConfig normal finish")
	return logical.RespondWithStatusCode(nil, req, http.StatusOK)
}
