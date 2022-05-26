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
			Pattern: path.Join("configure_extension", "flant_flow"),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleReadConfig,
					Summary:  "read flant_flow extension config",
				},
			},
		},
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
			Pattern: path.Join("configure_extension", "flant_flow", "all_flant_group", uuid.Pattern("all_flant_group_uuid")),
			Fields: map[string]*framework.FieldSchema{
				"all_flant_group_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a group which contains all flant teammates",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleConfigAllFlantGroup,
					Summary:  "Set all Flant teammates group uuid",
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleConfigAllFlantGroup,
					Summary:  "Set all Flant teammates group uuid",
				},
			},
		},
		{
			Pattern: path.Join("configure_extension", "flant_flow", "all_flant_group_roles"),
			Fields: map[string]*framework.FieldSchema{
				"roles": {
					Type:        framework.TypeStringSlice,
					Description: "names of global scoped roles to be set for all teammates",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleConfigAllFlantGroupRoles,
					Summary:  "Set all Flant teammates roles",
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleConfigAllFlantGroupRoles,
					Summary:  "Set all Flant teammates roles",
				},
			},
		},
		{
			Pattern: path.Join("configure_extension", "flant_flow", "role_rules", framework.GenericNameRegex("specific_team")+"$"),
			Fields: map[string]*framework.FieldSchema{
				"specific_team": {
					Type:        framework.TypeNameString,
					Description: fmt.Sprintf("Specific team type. Mandatory keys:%v", config.MandatoryRoleRulesForSpecificTeams),
					Required:    true,
				},
				"specific_roles": {
					Type:        framework.TypeStringSlice,
					Description: "Set of roles for specific team type. Passed roles will be used for automatically created rolebindings",
					Required:    true,
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
	b.Logger().Info("handleConfigFlantTenant started")
	defer b.Logger().Info("handleConfigFlantTenant exit")
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

func (b *flantFlowConfigureBackend) handleConfigAllFlantGroup(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("handleConfigAllFlantGroup started")
	defer b.Logger().Info("handleConfigAllFlantGroup exit")
	txn := b.storage.Txn(true)
	defer txn.Commit() //nolint:errcheck
	allFlantGroupUUID := data.Get("all_flant_group_uuid").(string)
	cfg, err := usecase.Config(txn).SetAllFlantGroupUUID(ctx, req.Storage, allFlantGroupUUID)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	b.setLiveConfig(cfg)
	b.Logger().Info("handleConfig normal finish")
	return logical.RespondWithStatusCode(nil, req, http.StatusOK)
}

func (b *flantFlowConfigureBackend) handleConfigAllFlantGroupRoles(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("handleConfigAllFlantGroupRoles  started")
	defer b.Logger().Info("handleConfigAllFlantGroupRoles exit")
	txn := b.storage.Txn(true)
	defer txn.Commit() //nolint:errcheck
	allFlantGroupRoles := data.Get("roles").([]string)
	cfg, err := usecase.Config(txn).SetAllFlantGroupRoles(ctx, req.Storage, allFlantGroupRoles)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	b.setLiveConfig(cfg)
	b.Logger().Info("handleConfig normal finish")
	return logical.RespondWithStatusCode(nil, req, http.StatusOK)
}

func (b *flantFlowConfigureBackend) handleConfigSpecificRoles(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("handleConfigSpecificRoles started")
	defer b.Logger().Info("handleConfigSpecificRoles exit")
	txn := b.storage.Txn(true)
	defer txn.Commit() //nolint:errcheck
	teamType := data.Get("specific_team").(string)
	roles := data.Get("specific_roles").([]string)
	if len(roles) == 0 {
		return backentutils.ResponseErr(req,
			fmt.Errorf("%w: mandatory param 'specific_roles' not passed, or is empty", consts.ErrInvalidArg))
	}
	cfg, err := usecase.Config(txn).UpdateSpecificRoles(ctx, req.Storage, teamType, roles)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	b.setLiveConfig(cfg)

	b.Logger().Info("handleConfig normal finish")
	return logical.RespondWithStatusCode(nil, req, http.StatusOK)
}

func (b *flantFlowConfigureBackend) handleConfigSpecificTeams(ctx context.Context, req *logical.Request,
	data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("handleConfigSpecificTeams started")
	defer b.Logger().Info("handleConfigSpecificTeams exit")
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

func (b *flantFlowConfigureBackend) handleReadConfig(ctx context.Context, req *logical.Request,
	_ *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("read flant_flow config started")
	defer b.Logger().Info("read flant_flow config")

	cfg := b.liveConfig
	resp := &logical.Response{
		Data: map[string]interface{}{
			"flant_flow_cfg": cfg,
		},
	}
	b.Logger().Info("read flant_flow config normal finish")
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
