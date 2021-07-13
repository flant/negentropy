package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	FeatureFlagType = "feature_flag" // also, memdb schema name
)

func FeatureFlagSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			FeatureFlagType: {
				Name: FeatureFlagType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
				},
			},
		},
	}
}

type FeatureFlag struct {
	Name FeatureFlagName `json:"name"` // PK
}

func (t *FeatureFlag) ObjType() string {
	return FeatureFlagType
}

func (t *FeatureFlag) ObjId() string {
	return t.Name
}

type FeatureFlagRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewFeatureFlagRepository(tx *io.MemoryStoreTxn) *FeatureFlagRepository {
	return &FeatureFlagRepository{tx}
}

func (r *FeatureFlagRepository) Create(ff *FeatureFlag) error {
	_, err := r.Get(ff.Name)
	if err == ErrNotFound {
		return r.db.Insert(FeatureFlagType, ff)
	}
	if err != nil {
		return err
	}
	return ErrAlreadyExists
}

func (r *FeatureFlagRepository) Get(name FeatureFlagName) (*FeatureFlag, error) {
	raw, err := r.db.First(FeatureFlagType, PK, name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*FeatureFlag), nil
}

func (r *FeatureFlagRepository) Delete(name FeatureFlagName) error {
	// TODO Cannot be deleted when in use by role, tenant, or project
	featureFlag, err := r.Get(name)
	if err != nil {
		return err
	}
	return r.db.Delete(FeatureFlagType, featureFlag)
}

func (r *FeatureFlagRepository) save(ff *FeatureFlag) error {
	return r.db.Insert(FeatureFlagType, ff)
}

func (r *FeatureFlagRepository) delete(name string) error {
	featureFlag, err := r.Get(name)
	if err != nil {
		return err
	}
	return r.db.Delete(FeatureFlagType, featureFlag)
}

func (r *FeatureFlagRepository) List() ([]FeatureFlagName, error) {
	iter, err := r.db.Get(FeatureFlagType, PK)
	if err != nil {
		return nil, err
	}

	list := []FeatureFlagName{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		ff := raw.(*FeatureFlag)
		list = append(list, ff.Name)
	}
	return list, nil
}

func (r *FeatureFlagRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.delete(objID)
	}

	ff := &FeatureFlag{}
	err := json.Unmarshal(data, ff)
	if err != nil {
		return err
	}

	return r.save(ff)
}

type TenantFeatureFlag struct {
	FeatureFlag           `json:",inline"`
	EnabledForNewProjects bool `json:"enabled_for_new"`
}

type TenantFeatureFlagRepository struct {
	tx *io.MemoryStoreTxn

	ffRepo     *FeatureFlagRepository
	tenantRepo *TenantRepository
	roleRepo   *RoleRepository
}

func NewTenantFeatureFlagRepository(tx *io.MemoryStoreTxn) *TenantFeatureFlagRepository {
	return &TenantFeatureFlagRepository{
		tx,
		NewFeatureFlagRepository(tx),
		NewTenantRepository(tx),
		NewRoleRepository(tx),
	}
}

func (t *TenantFeatureFlagRepository) GetFeatureFlags(tenantID string) ([]TenantFeatureFlag, error) {
	tenant, err := t.tenantRepo.GetByID(tenantID)
	if err != nil {
		return nil, err
	}

	return tenant.FeatureFlags, nil
}

func (t *TenantFeatureFlagRepository) AvailableRoles(tenantID string) ([]*Role, error) {
	tenant, err := t.tenantRepo.GetByID(tenantID)
	if err != nil {
		return nil, err
	}

	if len(tenant.FeatureFlags) == 0 {
		return nil, err
	}

	featureFlagsMap := make(map[FeatureFlagName]struct{}, len(tenant.FeatureFlags))
	for _, ff := range tenant.FeatureFlags {
		featureFlagsMap[ff.Name] = struct{}{}
	}

	available := make([]*Role, 0)

	err = t.roleRepo.Iter(func(role *Role) (bool, error) {
		for _, rf := range role.RequireOneOfFeatureFlags {
			if _, ok := featureFlagsMap[rf]; ok {
				available = append(available, role)
				break
			}
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return available, nil
}

func (t *TenantFeatureFlagRepository) SetFlagToTenant(tenantID string, featureFlag TenantFeatureFlag) (*Tenant, error) {
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

func (t *TenantFeatureFlagRepository) RemoveFlagFromTenant(tenantID string, featureFlagName string) (*Tenant, error) {
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

type ProjectFeatureFlagRepository struct {
	tx *io.MemoryStoreTxn

	ffRepo      *FeatureFlagRepository
	projectRepo *ProjectRepository
}

// Feature flag for projects
func NewProjectFeatureFlagRepository(tx *io.MemoryStoreTxn) *ProjectFeatureFlagRepository {
	return &ProjectFeatureFlagRepository{tx, NewFeatureFlagRepository(tx), NewProjectRepository(tx)}
}

func (t *ProjectFeatureFlagRepository) SetFlagToProject(tenantID, projectID string, featureFlag FeatureFlag) (*Project, error) {
	_, err := t.ffRepo.Get(featureFlag.Name)
	if err != nil {
		return nil, err
	}

	project, err := t.projectRepo.GetByID(projectID)
	if err != nil {
		return nil, err
	}
	if project.TenantUUID != tenantID {
		return nil, ErrNotFound
	}

	for _, pff := range project.FeatureFlags {
		if pff.Name == featureFlag.Name {
			return project, nil
		}
	}

	project.FeatureFlags = append(project.FeatureFlags, featureFlag)

	return project, t.projectRepo.Update(project)
}

func (t *ProjectFeatureFlagRepository) RemoveFlagFromProject(tenantID, projectID string, featureFlagName string) (*Project, error) {
	ff, err := t.ffRepo.Get(featureFlagName)
	if err != nil {
		return nil, err
	}

	project, err := t.projectRepo.GetByID(projectID)
	if err != nil {
		return nil, err
	}

	if project.TenantUUID != tenantID {
		return nil, ErrNotFound
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
