package authz

import (
	"fmt"
	"sort"

	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type EffectiveRoleChecker struct {
	TenantRepo    *iam_repo.TenantRepository
	ProjectRepo   *iam_repo.ProjectRepository
	RolesResolver iam_usecase.RoleResolver
	RoleRepo      *iam_repo.RoleRepository
}

func NewEffectiveRoleChecker(txn *io.MemoryStoreTxn) *EffectiveRoleChecker {
	return &EffectiveRoleChecker{
		TenantRepo:    iam_repo.NewTenantRepository(txn),
		ProjectRepo:   iam_repo.NewProjectRepository(txn),
		RolesResolver: iam_usecase.NewRoleResolver(txn),
		RoleRepo:      iam_repo.NewRoleRepository(txn),
	}
}

type EffectiveRoleResult struct {
	Role    iam_model.RoleName          `json:"role"`
	Tenants []EffectiveRoleTenantResult `json:"tenants"`
}

type EffectiveRoleTenantResult struct {
	TenantUUID       iam_model.TenantUUID         `json:"uuid"`
	TenantIdentifier string                       `json:"identifier"`
	TenantOptions    map[string][]interface{}     `json:"tenant_options,omitempty"`
	Projects         []EffectiveRoleProjectResult `json:"projects"`
}

type EffectiveRoleProjectResult struct {
	ProjectUUID       iam_model.ProjectUUID    `json:"uuid"`
	ProjectIdentifier string                   `json:"identifier"`
	ProjectOptions    map[string][]interface{} `json:"project_options,omitempty"`
	RequireMFA        bool                     `json:"require_mfa,omitempty"`
	NeedApprovals     bool                     `json:"need_approvals,omitempty"`
}

func (c *EffectiveRoleChecker) CheckEffectiveRoles(subject model.Subject, roles []iam_model.RoleName) ([]EffectiveRoleResult, error) {
	if subject.Type != iam_model.UserType {
		return nil, fmt.Errorf("%w: path available only for users", consts.ErrInvalidArg)
	}
	effectiveRoles, err := c.RolesResolver.CollectUserEffectiveRoles(subject.UUID, roles)
	if err != nil {
		return nil, fmt.Errorf("collectingEffectiveRoles: %w", err)
	}

	rolesResults := map[iam_model.RoleName]tenantsResultMap{}
	rolesResults, err = c.mapToEffectiveRoleResult(effectiveRoles)
	if err != nil {
		return nil, fmt.Errorf("mappingEffectiveRoles: %w", err)
	}
	return c.buildEffectiveRoleResults(roles, rolesResults)
}

type (
	projectsResultMap = map[iam_model.ProjectUUID]EffectiveRoleProjectResult
	tenantResult      struct {
		projects      projectsResultMap
		tenantOptions map[string][]interface{}
	}
	tenantsResultMap = map[iam_model.TenantUUID]tenantResult
)

func (c *EffectiveRoleChecker) buildEffectiveRoleResults(roleNames []iam_model.RoleName, rolesResults map[iam_model.RoleName]tenantsResultMap) ([]EffectiveRoleResult, error) {
	result := make([]EffectiveRoleResult, 0, len(rolesResults))
	for _, roleName := range roleNames {
		tenantsMap := rolesResults[roleName]
		tenantsResults, err := c.buildEffectiveRoleTenantsResults(tenantsMap)
		if err != nil {
			return nil, fmt.Errorf("building EffectiveRoleTenantResult: %w", err)
		}
		result = append(result, EffectiveRoleResult{
			Role:    roleName,
			Tenants: tenantsResults,
		})
	}
	return result, nil
}

func (c *EffectiveRoleChecker) buildEffectiveRoleTenantsResults(tenants tenantsResultMap) ([]EffectiveRoleTenantResult, error) {
	result := make([]EffectiveRoleTenantResult, 0, len(tenants))
	for tenantUUID, tenantResult := range tenants {
		tenant, err := c.TenantRepo.GetByID(tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("getting tenant by uuid=%s: %w", tenantUUID, err)
		}
		projectsResults, err := c.buildEffectiveRoleProjectsResults(tenantResult.projects)
		if err != nil {
			return nil, fmt.Errorf("building EffectiveRoleProjectResult: %w", err)
		}
		result = append(result, EffectiveRoleTenantResult{
			TenantUUID:       tenantUUID,
			TenantIdentifier: tenant.Identifier,
			Projects:         projectsResults,
			TenantOptions:    tenantResult.tenantOptions,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].TenantIdentifier < result[j].TenantIdentifier })
	return result, nil
}

func (c *EffectiveRoleChecker) buildEffectiveRoleProjectsResults(projects projectsResultMap) ([]EffectiveRoleProjectResult, error) {
	result := make([]EffectiveRoleProjectResult, 0, len(projects))
	for projectUUID, projectResult := range projects {
		project, err := c.ProjectRepo.GetByID(projectUUID)
		if err != nil {
			return nil, fmt.Errorf("getting project by uuid=%s: %w", projectUUID, err)
		}
		result = append(result, EffectiveRoleProjectResult{
			ProjectUUID:       projectUUID,
			ProjectIdentifier: project.Identifier,
			RequireMFA:        projectResult.RequireMFA,
			NeedApprovals:     projectResult.NeedApprovals,
			ProjectOptions:    projectResult.ProjectOptions,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ProjectIdentifier < result[j].ProjectIdentifier })
	return result, nil
}

func (c *EffectiveRoleChecker) mapToEffectiveRoleResult(effectiveRoles map[iam_model.RoleName][]iam_usecase.EffectiveRole) (map[iam_model.RoleName]tenantsResultMap, error) {
	result := map[iam_model.RoleName]tenantsResultMap{}
	for roleName, ers := range effectiveRoles {
		role, err := c.RoleRepo.GetByID(roleName)
		if err != nil {
			return nil, err
		}
		tenantsResults := tenantsResultMap{}
		for _, er := range ers {
			tenant, exist := tenantsResults[er.TenantUUID]
			if !exist {
				tenant = tenantResult{
					projects:      projectsResultMap{},
					tenantOptions: map[string][]interface{}{},
				}
			}
			isTenantOptions := role.Scope == iam_model.RoleScopeTenant || er.AnyProject
			if isTenantOptions {
				tenant.tenantOptions = concatenateOptions(tenant.tenantOptions, er.Options)
			}
			tenant.projects, err = c.enrichTenantResult(tenant.projects, er, !isTenantOptions)
			if err != nil {
				return nil, err
			}
			tenantsResults[er.TenantUUID] = tenant
		}
		result[roleName] = tenantsResults
	}
	return result, nil
}

func concatenateOptions(options map[string][]interface{}, erOptions map[string]interface{}) map[string][]interface{} {
	for k, v := range erOptions {
		values, keyExists := options[k]
		if keyExists {
			options[k] = append(values, v)
		} else {
			options[k] = []interface{}{v}
		}
	}
	return options
}

func (c *EffectiveRoleChecker) enrichTenantResult(projectsResults map[iam_model.ProjectUUID]EffectiveRoleProjectResult,
	er iam_usecase.EffectiveRole, isProjectOptions bool) (projectsResultMap, error) {
	projectsOfER := er.Projects
	var err error
	if er.AnyProject {
		projectsOfER, err = c.ProjectRepo.ListIDs(er.TenantUUID, false)
		if err != nil {
			return nil, fmt.Errorf("collecting all projects for tenant: %s: %w", er.TenantUUID, err)
		}
	}
	for _, projectUUID := range projectsOfER {
		effectiveRoleProjectResult, exists := projectsResults[projectUUID]
		if !exists {
			effectiveRoleProjectResult = EffectiveRoleProjectResult{
				ProjectUUID:    projectUUID,
				RequireMFA:     er.RequireMFA,
				NeedApprovals:  er.NeedApprovals > 0,
				ProjectOptions: map[string][]interface{}{},
			}
		} else {
			if !er.RequireMFA {
				effectiveRoleProjectResult.RequireMFA = false
			}
			if er.NeedApprovals == 0 {
				effectiveRoleProjectResult.NeedApprovals = false
			}
		}
		if isProjectOptions {
			effectiveRoleProjectResult.ProjectOptions = concatenateOptions(effectiveRoleProjectResult.ProjectOptions, er.Options)
		}
		projectsResults[projectUUID] = effectiveRoleProjectResult
	}
	return projectsResults, nil
}
