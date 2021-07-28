package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleService struct {
	db *io.MemoryStoreTxn
}

func Roles(db *io.MemoryStoreTxn) *RoleService {
	return &RoleService{db: db}
}

func (s *RoleService) Create(role *model.Role) error {
	return model.NewRoleRepository(s.db).Create(role)
}

func (s *RoleService) Get(roleID model.RoleName) (*model.Role, error) {
	return model.NewRoleRepository(s.db).GetByID(roleID)
}

func (s *RoleService) List(showArchived bool) ([]*model.Role, error) {
	return model.NewRoleRepository(s.db).List(showArchived)
}

func (s *RoleService) Update(updated *model.Role) error {
	repo := model.NewRoleRepository(s.db)

	stored, err := repo.GetByID(updated.Name)
	if err != nil {
		return err
	}

	updated.Scope = stored.Scope // type cannot be changed

	// TODO validate feature flags: role must not become unaccessible in the scope where it is used
	// TODO forbid backwards-incompatible changes of the options schema

	return repo.Update(updated)
}

func (s *RoleService) Delete(roleID model.RoleName, archivingTimestamp model.UnixTime, archivingHash int64) error {
	// TODO before the deletion, check it is not used in
	//      * role_biondings,
	//      * approvals,
	//      * tokens,
	//      * service account passwords
	// TODO REMOVE FROM archived
	//      * role_biondings,
	//      * approvals,
	//      * tokens,
	//      * service account passwords
	return model.NewRoleRepository(s.db).Delete(roleID, archivingTimestamp, archivingHash)
}

func (s *RoleService) Include(roleID model.RoleName, subRole *model.IncludedRole) error {
	repo := model.NewRoleRepository(s.db)

	// validate target exists
	role, err := repo.GetByID(roleID)
	if err != nil {
		return err
	}

	// validate source exists
	if _, err := repo.GetByID(subRole.Name); err != nil {
		return err
	}

	// TODO validate the template

	includeRole(role, subRole)

	return repo.Update(role)
}

func (s *RoleService) Exclude(roleID, exclRoleID model.RoleName) error {
	repo := model.NewRoleRepository(s.db)

	role, err := repo.GetByID(roleID)
	if err != nil {
		return err
	}

	excludeRole(role, exclRoleID)

	return repo.Update(role)
}

func includeRole(role *model.Role, subRole *model.IncludedRole) {
	for i, present := range role.IncludedRoles {
		if present.Name == subRole.Name {
			role.IncludedRoles[i] = *subRole
			return
		}
	}

	role.IncludedRoles = append(role.IncludedRoles, *subRole)
}

func excludeRole(role *model.Role, exclRoleID model.RoleName) {
	var i int
	var ir model.IncludedRole
	var found bool

	for i, ir = range role.IncludedRoles {
		found = ir.Name == exclRoleID
		if found {
			break
		}
	}

	if !found {
		return
	}

	cleaned := make([]model.IncludedRole, len(role.IncludedRoles)-1)
	copy(cleaned, role.IncludedRoles[:i])
	copy(cleaned[i:], role.IncludedRoles[i+1:])

	role.IncludedRoles = cleaned
}
