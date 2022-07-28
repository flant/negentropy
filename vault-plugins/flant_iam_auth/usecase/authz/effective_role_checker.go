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
}

func NewEffectiveRoleChecker(txn *io.MemoryStoreTxn) *EffectiveRoleChecker {
	return &EffectiveRoleChecker{
		TenantRepo:    iam_repo.NewTenantRepository(txn),
		ProjectRepo:   iam_repo.NewProjectRepository(txn),
		RolesResolver: iam_usecase.NewRoleResolver(txn),
	}
}

type EffectiveRoleResult struct {
	Role    iam_model.RoleName          `json:"role"`
	Tenants []EffectiveRoleTenantResult `json:"tenants,omitempty"`
}

type EffectiveRoleTenantResult struct {
	TenantUUID       iam_model.TenantUUID         `json:"uuid"`
	TenantIdentifier string                       `json:"identifier"`
	Projects         []EffectiveRoleProjectResult `json:"projects,omitempty"`
}

type EffectiveRoleProjectResult struct {
	ProjectUUID       iam_model.ProjectUUID `json:"uuid"`
	ProjectIdentifier string                `json:"identifier"`
	RequireMFA        bool                  `json:"require_mfa,omitempty"`
	NeedApprovals     bool                  `json:"need_approvals,omitempty"`
}

func (c *EffectiveRoleChecker) CheckEffectiveRoles(subject model.Subject, roles []iam_model.RoleName) ([]EffectiveRoleResult, error) {
	if subject.Type != iam_model.UserType {
		return nil, fmt.Errorf("%w: path available only for users", consts.ErrInvalidArg)
	}
	effectiveRoles, err := c.RolesResolver.CollectUserEffectiveRoles(subject.UUID, roles)
	if err != nil {
		return nil, fmt.Errorf("collectingEffectiveRoles: %w", err)
	}

	var rolesResults = map[iam_model.RoleName]tenantsResultMap{}
	rolesResults, err = c.mapToEffectiveRoleResult(effectiveRoles)
	if err != nil {
		return nil, fmt.Errorf("mappingEffectiveRoles: %w", err)
	}
	return c.buildEffectiveRoleResults(rolesResults)
}

type projectsResultMap = map[iam_model.ProjectUUID]EffectiveRoleProjectResult
type tenantsResultMap = map[iam_model.TenantUUID]projectsResultMap

func (c *EffectiveRoleChecker) buildEffectiveRoleResults(rolesResults map[iam_model.RoleName]tenantsResultMap) ([]EffectiveRoleResult, error) {
	result := make([]EffectiveRoleResult, 0, len(rolesResults))
	for roleName, tenantsMap := range rolesResults {
		tenantsResults, err := c.buildEffectiveRoleTenantsResults(tenantsMap)
		if err != nil {
			return nil, fmt.Errorf("building EffectiveRoleTenantResult: %w", err)
		}
		result = append(result, EffectiveRoleResult{
			Role:    roleName,
			Tenants: tenantsResults,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Role < result[j].Role })
	return result, nil
}

func (c *EffectiveRoleChecker) buildEffectiveRoleTenantsResults(tenants tenantsResultMap) ([]EffectiveRoleTenantResult, error) {
	result := make([]EffectiveRoleTenantResult, 0, len(tenants))
	for tenantUUID, projectsMap := range tenants {
		tenant, err := c.TenantRepo.GetByID(tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("getting tenant by uuid=%s: %w", tenantUUID, err)
		}
		projectsResults, err := c.buildEffectiveRoleProjectsResults(projectsMap)
		if err != nil {
			return nil, fmt.Errorf("building EffectiveRoleProjectResult: %w", err)
		}
		result = append(result, EffectiveRoleTenantResult{
			TenantUUID:       tenantUUID,
			TenantIdentifier: tenant.Identifier,
			Projects:         projectsResults,
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
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ProjectIdentifier < result[j].ProjectIdentifier })
	return result, nil
}

func (c *EffectiveRoleChecker) mapToEffectiveRoleResult(effectiveRoles map[iam_model.RoleName][]iam_usecase.EffectiveRole) (map[iam_model.RoleName]tenantsResultMap, error) {
	result := map[iam_model.RoleName]tenantsResultMap{}
	var err error
	for roleName, ers := range effectiveRoles {
		tenantsResults := tenantsResultMap{}
		for _, er := range ers {
			projectsResuts, exist := tenantsResults[er.TenantUUID]
			if !exist {
				projectsResuts = projectsResultMap{}
			}
			projectsResuts, err = c.enrichTenantResult(projectsResuts, er)
			if err != nil {
				return nil, err
			}
		}
		result[roleName] = tenantsResults
	}
	return result, nil
}

func (c *EffectiveRoleChecker) enrichTenantResult(projectsResults map[iam_model.ProjectUUID]EffectiveRoleProjectResult,
	er iam_usecase.EffectiveRole) (projectsResultMap, error) {
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
				ProjectUUID:   projectUUID,
				RequireMFA:    er.RequireMFA,
				NeedApprovals: er.NeedApprovals > 0,
			}
		} else {
			if !er.RequireMFA {
				effectiveRoleProjectResult.RequireMFA = false
			}
			if er.NeedApprovals == 0 {
				effectiveRoleProjectResult.NeedApprovals = false
			}
		}
		projectsResults[projectUUID] = effectiveRoleProjectResult
	}
	return projectsResults, nil
}
