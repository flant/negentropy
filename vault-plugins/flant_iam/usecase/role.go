package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type RoleService struct {
	db *io.MemoryStoreTxn
}

func Roles(db *io.MemoryStoreTxn) *RoleService {
	return &RoleService{db: db}
}

func (s *RoleService) Create(role *model.Role) error {
	repo := repo.NewRoleRepository(s.db)
	_, err := repo.GetByID(role.Name)
	if !errors.Is(err, consts.ErrNotFound) {
		return fmt.Errorf("%w: %s", consts.ErrAlreadyExists, role.Name)
	}
	if err != nil && !errors.Is(err, consts.ErrNotFound) {
		return err
	}
	if err := role.ValidateScope(); err != nil {
		return fmt.Errorf("%w: %s", consts.ErrInvalidArg, err.Error())
	}
	if err := checkRoleOptionSchema(role.OptionsSchema); err != nil {
		return fmt.Errorf("%w: %s", consts.ErrInvalidArg, err.Error())
	}
	return repo.Create(role)
}

func (s *RoleService) Get(roleID model.RoleName) (*model.Role, error) {
	return repo.NewRoleRepository(s.db).GetByID(roleID)
}

func (s *RoleService) List(showArchived bool) ([]*model.Role, error) {
	return repo.NewRoleRepository(s.db).List(showArchived)
}

func (s *RoleService) Update(updated *model.Role) error {
	repo := repo.NewRoleRepository(s.db)

	stored, err := repo.GetByID(updated.Name)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}

	updated.Scope = stored.Scope                         // type cannot be changed
	updated.TenantIsOptional = stored.TenantIsOptional   // type cannot be changed
	updated.ProjectIsOptional = stored.ProjectIsOptional // type cannot be changed

	// TODO validate feature flags: role must not become unaccessible in the scope where it is used
	if err := checkBackwardsCompatibility(stored.OptionsSchema, updated.OptionsSchema); err != nil {
		return err
	}

	return repo.Update(updated)
}

func (s *RoleService) Delete(roleID model.RoleName) error {
	// TODO before the deletion, check it is not used in
	//      * role_biondings,
	//      * approvals,
	//      * tokens,
	//      * service account passwords
	// TODO - REMOVE FROM archived
	//      * role_biondings,
	//      * approvals,
	//      * tokens,
	//      * service account passwords
	return repo.NewRoleRepository(s.db).Delete(roleID, memdb.NewArchiveMark())
}

func (s *RoleService) Include(roleID model.RoleName, subRole *model.IncludedRole) error {
	repo := repo.NewRoleRepository(s.db)

	// validate target exists
	role, err := repo.GetByID(roleID)
	if err != nil {
		return err
	}
	if role.Archived() {
		return consts.ErrIsArchived
	}

	// validate source exists
	sub, err := repo.GetByID(subRole.Name)
	if err != nil {
		return err
	}

	if role.ForbinddenDirectUse && !sub.ForbinddenDirectUse {
		return fmt.Errorf("target role is fordidden for using in rolebindings, can't contain allowed for rolebinding role: %#v", sub)
	}

	// TODO validate the template

	includeRole(role, subRole)

	return repo.Update(role)
}

func (s *RoleService) Exclude(roleID, exclRoleID model.RoleName) error {
	repo := repo.NewRoleRepository(s.db)

	role, err := repo.GetByID(roleID)
	if err != nil {
		return err
	}
	if role.Archived() {
		return consts.ErrIsArchived
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
