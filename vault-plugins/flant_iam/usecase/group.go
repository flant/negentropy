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
	tenantUUID model.TenantUUID

	repo            *model.GroupRepository
	tenantsRepo     *model.TenantRepository
	subjectsFetcher *SubjectsFetcher
}

func Groups(db *io.MemoryStoreTxn, tid model.TenantUUID) *GroupsService {
	return &GroupsService{
		tenantUUID: tid,

		repo:            model.NewGroupRepository(db),
		tenantsRepo:     model.NewTenantRepository(db),
		subjectsFetcher: NewSubjectsFetcher(db),
	}
}

func (s *GroupsService) Create(group *model.Group) error {
	tenant, err := s.tenantsRepo.GetByID(s.tenantUUID)
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

	subj, err := s.subjectsFetcher.Fetch(group.Subjects)
	if err != nil {
		return err
	}
	group.Groups = subj.Groups
	group.ServiceAccounts = subj.ServiceAccounts
	group.Users = subj.Users

	return s.repo.Create(group)
}

func (s *GroupsService) Update(group *model.Group) error {
	stored, err := s.repo.GetByID(group.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != s.tenantUUID {
		return model.ErrNotFound
	}
	if stored.Origin != group.Origin {
		return model.ErrBadOrigin
	}
	if stored.Version != group.Version {
		return model.ErrBadVersion
	}

	tenant, err := s.tenantsRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}
	// Update
	group.TenantUUID = s.tenantUUID
	group.Version = model.NewResourceVersion()
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	subj, err := s.subjectsFetcher.Fetch(group.Subjects)
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

	return s.repo.Update(group)
}

/*
TODO Clean from everywhere:
	* other groups
	* role_bindings
	* approvals
	* identity_sharings
*/
func (s *GroupsService) Delete(origin model.ObjectOrigin, id model.GroupUUID) error {
	group, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if group.Origin != origin {
		return model.ErrBadOrigin
	}
	return s.repo.Delete(id)
}

func (s *GroupsService) SetExtension(ext *model.Extension) error {
	obj, err := s.repo.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[model.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return s.repo.Update(obj)
}

func (s *GroupsService) UnsetExtension(origin model.ObjectOrigin, uuid model.GroupUUID) error {
	obj, err := s.repo.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return s.repo.Update(obj)
}

func (s *GroupsService) List(tid model.TenantUUID) ([]*model.Group, error) {
	return s.repo.List(tid)
}
