package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

// FeatureFlagService manages global list of feature_flags
type FeatureFlagService struct {
	db *io.MemoryStoreTxn
}

func Featureflags(db *io.MemoryStoreTxn) *FeatureFlagService {
	return &FeatureFlagService{db: db}
}

func (s *FeatureFlagService) Create(featureFlag *model.FeatureFlag) error {
	return repo.NewFeatureFlagRepository(s.db).Create(featureFlag)
}

func (s *FeatureFlagService) List(showArchived bool) ([]*model.FeatureFlag, error) {
	return repo.NewFeatureFlagRepository(s.db).List(showArchived)
}

func (s *FeatureFlagService) Delete(id model.FeatureFlagName) error {
	// TODO before the deletion, check
	// TODO - REMOVE FROM archived
	return repo.NewFeatureFlagRepository(s.db).Delete(id, memdb.NewArchiveMark())
}

// TenantFeatureFlagService manages feature_flags for tenants
type TenantFeatureFlagService struct {
	tenantUUID model.TenantUUID
	ffRepo     *repo.FeatureFlagRepository
	tenantRepo *repo.TenantRepository
	roleRepo   *repo.RoleRepository
}

func TenantFeatureFlags(tx *io.MemoryStoreTxn, id model.TenantUUID) *TenantFeatureFlagService {
	return &TenantFeatureFlagService{
		tenantUUID: id,
		ffRepo:     repo.NewFeatureFlagRepository(tx),
		tenantRepo: repo.NewTenantRepository(tx),
		roleRepo:   repo.NewRoleRepository(tx),
	}
}

func (s *TenantFeatureFlagService) List() ([]model.TenantFeatureFlag, error) {
	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return nil, err
	}

	return tenant.FeatureFlags, nil
}

func (s *TenantFeatureFlagService) AvailableRoles() ([]*model.Role, error) {
	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
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

	err = s.roleRepo.Iter(func(role *model.Role) (bool, error) {
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

func (s *TenantFeatureFlagService) Add(featureFlag model.TenantFeatureFlag) (*model.Tenant, error) {
	_, err := s.ffRepo.GetByID(featureFlag.Name)
	if err != nil {
		return nil, err
	}

	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return nil, err
	}

	for _, tff := range tenant.FeatureFlags {
		if tff.Name == featureFlag.Name {
			// update
			tff.EnabledForNewProjects = featureFlag.EnabledForNewProjects
			return tenant, s.tenantRepo.Update(tenant)
		}
	}

	tenant.FeatureFlags = append(tenant.FeatureFlags, featureFlag)

	return tenant, s.tenantRepo.Update(tenant)
}

func (s *TenantFeatureFlagService) Delete(featureFlagName string) (*model.Tenant, error) {
	ff, err := s.ffRepo.GetByID(featureFlagName)
	if err != nil {
		return nil, err
	}

	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return nil, err
	}

	// TODO remove feature_flag from all nested projects
	// TODO: deny deleting if role become inaccessible

	for i, tff := range tenant.FeatureFlags {
		if tff.Name == ff.Name {
			tenant.FeatureFlags = append(tenant.FeatureFlags[:i], tenant.FeatureFlags[i+1:]...)
			// update
			return tenant, s.tenantRepo.Update(tenant)
		}
	}

	return tenant, nil
}

// ProjectFeatureFlagService manages feature flags for projects
type ProjectFeatureFlagService struct {
	tenantUUID  model.TenantUUID
	projectUUID model.ProjectUUID

	ffRepo      *repo.FeatureFlagRepository
	projectRepo *repo.ProjectRepository
}

// Feature flag for projects
func ProjectFeatureFlags(tx *io.MemoryStoreTxn, tenantID model.TenantUUID, projectID model.ProjectUUID) *ProjectFeatureFlagService {
	return &ProjectFeatureFlagService{
		tenantUUID:  tenantID,
		projectUUID: projectID,

		ffRepo:      repo.NewFeatureFlagRepository(tx),
		projectRepo: repo.NewProjectRepository(tx),
	}
}

func (s *ProjectFeatureFlagService) Add(featureFlagName model.FeatureFlagName) (*model.Project, error) {
	_, err := s.ffRepo.GetByID(featureFlagName)
	if err != nil {
		return nil, err
	}

	project, err := s.projectRepo.GetByID(s.projectUUID)
	if err != nil {
		return nil, err
	}
	if project.TenantUUID != s.tenantUUID {
		return nil, consts.ErrNotFound
	}

	for _, pff := range project.FeatureFlags {
		if pff == featureFlagName {
			return project, nil
		}
	}

	project.FeatureFlags = append(project.FeatureFlags, featureFlagName)

	return project, s.projectRepo.Update(project)
}

func (s *ProjectFeatureFlagService) Delete(featureFlagName model.FeatureFlagName) (*model.Project, error) {
	ff, err := s.ffRepo.GetByID(featureFlagName)
	if err != nil {
		return nil, err
	}

	project, err := s.projectRepo.GetByID(s.projectUUID)
	if err != nil {
		return nil, err
	}

	if project.TenantUUID != s.tenantUUID {
		return nil, consts.ErrNotFound
	}

	for i, pff := range project.FeatureFlags {
		if pff == ff.Name {
			project.FeatureFlags = append(project.FeatureFlags[:i], project.FeatureFlags[i+1:]...)
			// update
			return project, s.projectRepo.Update(project)
		}
	}

	return project, nil
}
