package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// TenantFeatureFlagManager manages feature flags for tenants
type TenantFeatureFlagManager struct {
	ffRepo     *model.FeatureFlagRepository
	tenantRepo *model.TenantRepository
	roleRepo   *model.RoleRepository
}

func TenantFeatureFlags(tx *io.MemoryStoreTxn) *TenantFeatureFlagManager {
	return &TenantFeatureFlagManager{
		ffRepo:     model.NewFeatureFlagRepository(tx),
		tenantRepo: model.NewTenantRepository(tx),
		roleRepo:   model.NewRoleRepository(tx),
	}
}

func (t *TenantFeatureFlagManager) List(tenantID string) ([]model.TenantFeatureFlag, error) {
	tenant, err := t.tenantRepo.GetByID(tenantID)
	if err != nil {
		return nil, err
	}

	return tenant.FeatureFlags, nil
}

func (t *TenantFeatureFlagManager) AvailableRoles(tenantID string) ([]*model.Role, error) {
	tenant, err := t.tenantRepo.GetByID(tenantID)
	if err != nil {
		return nil, err
	}

	if len(tenant.FeatureFlags) == 0 {
		return nil, err
	}

	featureFlagsMap := make(map[model.FeatureFlagName]struct{}, len(tenant.FeatureFlags))
	for _, ff := range tenant.FeatureFlags {
		featureFlagsMap[ff.Name] = struct{}{}
	}

	roles := make([]*model.Role, 0)

	err = t.roleRepo.Iter(func(role *model.Role) (bool, error) {
		for _, rf := range role.RequireOneOfFeatureFlags {
			if _, ok := featureFlagsMap[rf]; ok {
				roles = append(roles, role)
				break
			}
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return roles, nil
}

func (t *TenantFeatureFlagManager) Add(tenantID string, featureFlag model.TenantFeatureFlag) (*model.Tenant, error) {
	_, err := t.ffRepo.Get(featureFlag.Name)
	if err != nil {
		return nil, err
	}

	tenant, err := t.tenantRepo.GetByID(tenantID)
	if err != nil {
		return nil, err
	}

	for _, tff := range tenant.FeatureFlags {
		if tff.Name == featureFlag.Name {
			// update
			tff.EnabledForNewProjects = featureFlag.EnabledForNewProjects
			return tenant, t.tenantRepo.Update(tenant)
		}
	}

	tenant.FeatureFlags = append(tenant.FeatureFlags, featureFlag)

	return tenant, t.tenantRepo.Update(tenant)
}

func (t *TenantFeatureFlagManager) Delete(tenantID string, featureFlagName string) (*model.Tenant, error) {
	ff, err := t.ffRepo.Get(featureFlagName)
	if err != nil {
		return nil, err
	}

	tenant, err := t.tenantRepo.GetByID(tenantID)
	if err != nil {
		return nil, err
	}

	// TODO remove feature_flag from all nested projects
	// TODO: deny deleting if role become inaccessible

	for i, tff := range tenant.FeatureFlags {
		if tff.Name == ff.Name {
			tenant.FeatureFlags = append(tenant.FeatureFlags[:i], tenant.FeatureFlags[i+1:]...)
			// update
			return tenant, t.tenantRepo.Update(tenant)
		}
	}

	return tenant, nil
}

// ProjectFeatureFlagManager manages feature flags for projects
type ProjectFeatureFlagManager struct {
	ffRepo      *model.FeatureFlagRepository
	projectRepo *model.ProjectRepository
}

// Feature flag for projects
func ProjectFeatureFlags(tx *io.MemoryStoreTxn) *ProjectFeatureFlagManager {
	return &ProjectFeatureFlagManager{
		ffRepo:      model.NewFeatureFlagRepository(tx),
		projectRepo: model.NewProjectRepository(tx),
	}
}

func (t *ProjectFeatureFlagManager) Add(tenantID, projectID string, featureFlag model.FeatureFlag) (*model.Project, error) {
	_, err := t.ffRepo.Get(featureFlag.Name)
	if err != nil {
		return nil, err
	}

	project, err := t.projectRepo.GetByID(projectID)
	if err != nil {
		return nil, err
	}
	if project.TenantUUID != tenantID {
		return nil, model.ErrNotFound
	}

	for _, pff := range project.FeatureFlags {
		if pff.Name == featureFlag.Name {
			return project, nil
		}
	}

	project.FeatureFlags = append(project.FeatureFlags, featureFlag)

	return project, t.projectRepo.Update(project)
}

func (t *ProjectFeatureFlagManager) Delete(tenantID, projectID string, featureFlagName string) (*model.Project, error) {
	ff, err := t.ffRepo.Get(featureFlagName)
	if err != nil {
		return nil, err
	}

	project, err := t.projectRepo.GetByID(projectID)
	if err != nil {
		return nil, err
	}

	if project.TenantUUID != tenantID {
		return nil, model.ErrNotFound
	}

	for i, pff := range project.FeatureFlags {
		if pff.Name == ff.Name {
			project.FeatureFlags = append(project.FeatureFlags[:i], project.FeatureFlags[i+1:]...)
			// update
			return project, t.projectRepo.Update(project)
		}
	}

	return project, nil
}
