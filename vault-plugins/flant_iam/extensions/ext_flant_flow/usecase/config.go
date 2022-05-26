package usecase

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type ConfigService struct {
	configProvider  config.MutexedConfigManager
	tenantRepo      *iam_repo.TenantRepository
	groupRepo       *iam_repo.GroupRepository
	teamRepo        *repo.TeamRepository
	roleRepo        *iam_repo.RoleRepository
	roleBindingRepo *iam_repo.RoleBindingRepository
}

func Config(db *io.MemoryStoreTxn) *ConfigService {
	return &ConfigService{
		configProvider:  config.MutexedConfigManager{},
		tenantRepo:      iam_repo.NewTenantRepository(db),
		groupRepo:       iam_repo.NewGroupRepository(db),
		teamRepo:        repo.NewTeamRepository(db),
		roleRepo:        iam_repo.NewRoleRepository(db),
		roleBindingRepo: iam_repo.NewRoleBindingRepository(db),
	}
}

func (c *ConfigService) SetFlantTenantUUID(ctx context.Context, storage logical.Storage,
	flantUUID iam_model.TenantUUID) (*config.FlantFlowConfig, error) {
	cfg, err := c.configProvider.GetConfig(ctx, storage)
	if err != nil {
		return nil, err
	}
	if cfg.FlantTenantUUID != "" {
		return nil, fmt.Errorf("%w:flant_tenant_uuid is already set", consts.ErrInvalidArg)
	}
	if err = c.tenantRepo.Create(&iam_model.Tenant{
		UUID:       flantUUID,
		Version:    uuid.New(),
		Identifier: "flant",
		Language:   "english",
		Origin:     consts.OriginFlantFlowPredefined,
	}); err != nil {
		return nil, fmt.Errorf("%w:%s", err, flantUUID)
	}
	return c.configProvider.SetFlantTenantUUID(ctx, storage, flantUUID)
}

func (c *ConfigService) SetAllFlantGroupUUID(ctx context.Context, storage logical.Storage,
	allFlantGroupUUID iam_model.GroupUUID) (*config.FlantFlowConfig, error) {
	cfg, err := c.configProvider.GetConfig(ctx, storage)
	if err != nil {
		return nil, err
	}
	if cfg.FlantTenantUUID == "" {
		return nil, fmt.Errorf("%w:flant_tenant_uuid should be set first", consts.ErrNotConfigured)
	}
	if cfg.AllFlantGroupUUID != "" {
		return nil, fmt.Errorf("%w:all_flant_group_uuid is already set", consts.ErrInvalidArg)
	}
	err = c.groupRepo.Create(&iam_model.Group{
		UUID:           allFlantGroupUUID,
		TenantUUID:     cfg.FlantTenantUUID,
		Version:        uuid.New(),
		Identifier:     "all",
		FullIdentifier: "all@group.flant",
		Origin:         consts.OriginFlantFlowPredefined,
	})
	if err != nil {
		return nil, fmt.Errorf("%w:%s", err, allFlantGroupUUID)
	}

	return c.configProvider.SetAllFlantGroupUUID(ctx, storage, allFlantGroupUUID)
}

func (c *ConfigService) SetAllFlantGroupRoles(ctx context.Context, storage logical.Storage,
	allFlantGroupGlobalRoles []iam_model.RoleName) (*config.FlantFlowConfig, error) {
	cfg, err := c.configProvider.GetConfig(ctx, storage)
	if err != nil {
		return nil, err
	}
	if cfg.FlantTenantUUID == "" {
		return nil, fmt.Errorf("%w:flant_tenant_uuid should be set first", consts.ErrNotConfigured)
	}
	if cfg.AllFlantGroupUUID == "" {
		return nil, fmt.Errorf("%w:all_flant_group_uuid should be set first", consts.ErrNotConfigured)
	}

	roles := make([]iam_model.BoundRole, 0, len(allFlantGroupGlobalRoles))
	for _, role := range allFlantGroupGlobalRoles {
		roles = append(roles, iam_model.BoundRole{Name: role})
	}

	rbUUID := uuid.New()

	if cfg.AllFlantGroupRoleBindingUUID != "" {
		rbUUID = cfg.AllFlantGroupRoleBindingUUID
	}

	cfg, err = c.configProvider.SetAllFlantGroupRoles(ctx, storage, allFlantGroupGlobalRoles, rbUUID)
	if err != nil {
		return nil, fmt.Errorf("set all-flant rolebinding uuid: %w", err)
	}

	err = c.roleBindingRepo.Create(&iam_model.RoleBinding{
		UUID:            rbUUID,
		TenantUUID:      cfg.FlantTenantUUID,
		Version:         uuid.New(),
		Description:     "all-flant group rolebinding",
		ValidTill:       0,
		Groups:          []iam_model.GroupUUID{cfg.AllFlantGroupUUID},
		ServiceAccounts: nil,
		Members: []iam_model.MemberNotation{{
			Type: iam_model.GroupType,
			UUID: cfg.AllFlantGroupUUID,
		}},
		Roles:  roles,
		Origin: consts.OriginFlantFlow,
	})
	return cfg, err
}

func (c *ConfigService) SetPrimaryAdministratorsRoles(ctx context.Context, storage logical.Storage,
	roles []iam_model.RoleName) (*config.FlantFlowConfig, error) {
	cfg, err := c.configProvider.GetConfig(ctx, storage)
	if err != nil {
		return nil, err
	}
	if cfg.FlantTenantUUID == "" {
		return nil, fmt.Errorf("%w:flant_tenant_uuid should be set first", consts.ErrNotConfigured)
	}
	return c.configProvider.SetPrimaryAdministratorsRoles(ctx, storage, roles)
}

func (c *ConfigService) UpdateSpecificRoles(ctx context.Context, storage logical.Storage, teamType config.SpecializedTeam,
	roles []iam_model.RoleName) (*config.FlantFlowConfig, error) {
	for _, roleName := range roles {
		if _, err := c.roleRepo.GetByID(roleName); err != nil {
			return nil, fmt.Errorf("%w:%s", err, roleName)
		}
	}
	return c.configProvider.UpdateSpecificRoleRules(ctx, storage, teamType, roles)
}

func (c *ConfigService) UpdateSpecificTeams(ctx context.Context, storage logical.Storage, teamsMap map[string]string) (*config.FlantFlowConfig, error) {
	for _, teamUUID := range teamsMap {
		if _, err := c.teamRepo.GetByID(teamUUID); err != nil {
			return nil, fmt.Errorf("%w:%s", err, teamUUID)
		}
	}
	return c.configProvider.UpdateSpecificTeams(ctx, storage, teamsMap)
}

func (c *ConfigService) GetConfig(ctx context.Context, storage logical.Storage) (*config.FlantFlowConfig, error) {
	return c.configProvider.GetConfig(ctx, storage)
}
