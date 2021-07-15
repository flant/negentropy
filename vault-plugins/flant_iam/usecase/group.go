package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// generic: <identifier>@group.<tenant_identifier>
// builtin: <identifier>@<builtin_group_type>.group.<tenant_identifier>
func CalcGroupFullIdentifier(g *model.Group, tenant *model.Tenant) string {
	name := g.Identifier
	domain := "group." + tenant.Identifier
	return name + "@" + domain
}

type GroupsService struct {
	db *io.MemoryStoreTxn
}

func Groups(tx *io.MemoryStoreTxn) *GroupsService {
	return &GroupsService{tx}
}

func (s *GroupsService) Create(group *model.Group) error {
	tenant, err := model.NewTenantRepository(s.db).GetByID(group.TenantUUID)
	if err != nil {
		return err
	}

	if group.Version != "" {
		return model.ErrBadVersion
	}
	if group.Origin == "" {
		return model.ErrBadOrigin
	}
	group.Version = model.NewResourceVersion()
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	subj, err := NewSubjectsFetcher(s.db).Fetch(group.Subjects)
	if err != nil {
		return err
	}
	group.Groups = subj.Groups
	group.ServiceAccounts = subj.ServiceAccounts
	group.Users = subj.Users

	return model.NewGroupRepository(s.db).Create(group)
}

func (s *GroupsService) Update(group *model.Group) error {
	stored, err := model.NewGroupRepository(s.db).GetByID(group.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != group.TenantUUID {
		return model.ErrNotFound
	}
	if stored.Origin != group.Origin {
		return model.ErrBadOrigin
	}
	if stored.Version != group.Version {
		return model.ErrBadVersion
	}
	group.Version = model.NewResourceVersion()

	// Update

	tenant, err := model.NewTenantRepository(s.db).GetByID(group.TenantUUID)
	if err != nil {
		return err
	}
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	subj, err := NewSubjectsFetcher(s.db).Fetch(group.Subjects)
	if err != nil {
		return err
	}
	group.Groups = subj.Groups
	group.ServiceAccounts = subj.ServiceAccounts
	group.Users = subj.Users

	// Preserve fields, that are not always accessible from the outside, e.g. from HTTP API
	if group.Extensions == nil {
		group.Extensions = stored.Extensions
	}

	return model.NewGroupRepository(s.db).Update(group)
}

/*
TODO Clean from everywhere:
	* other groups
	* role_bindings
	* approvals
	* identity_sharings
*/
func (s *GroupsService) Delete(origin model.ObjectOrigin, id model.GroupUUID) error {
	repo := model.NewGroupRepository(s.db)
	group, err := repo.GetByID(id)
	if err != nil {
		return err
	}
	if group.Origin != origin {
		return model.ErrBadOrigin
	}
	return repo.Delete(id)
}

func (s *GroupsService) DeleteByTenant(tenantUUID model.TenantUUID) error {
	// TODO clean from parent groups
	_, err := s.db.DeleteAll(model.GroupType, model.TenantForeignPK, tenantUUID)
	return err
}

func (s *GroupsService) SetExtension(ext *model.Extension) error {
	repo := model.NewGroupRepository(s.db)
	obj, err := repo.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[model.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return repo.Update(obj)
}

func (s *GroupsService) UnsetExtension(origin model.ObjectOrigin, uuid model.GroupUUID) error {
	repo := model.NewGroupRepository(s.db)
	obj, err := repo.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return repo.Update(obj)
}
