package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

// generic: <identifier>@group.<tenant_identifier>
// builtin: <identifier>@<builtin_group_type>.group.<tenant_identifier>
func CalcGroupFullIdentifier(g *model.Group, tenant *model.Tenant) string {
	name := g.Identifier
	domain := "group." + tenant.Identifier
	return name + "@" + domain
}

type GroupService struct {
	tenantUUID model.TenantUUID

	repo           *iam_repo.GroupRepository
	tenantsRepo    *iam_repo.TenantRepository
	membersFetcher *MembersFetcher
}

func Groups(db *io.MemoryStoreTxn, tid model.TenantUUID) *GroupService {
	return &GroupService{
		tenantUUID: tid,

		repo:           iam_repo.NewGroupRepository(db),
		tenantsRepo:    iam_repo.NewTenantRepository(db),
		membersFetcher: NewMembersFetcher(db),
	}
}

func (s *GroupService) Create(group *model.Group) error {
	tenant, err := s.tenantsRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	if group.Version != "" {
		return consts.ErrBadVersion
	}
	if group.Origin == "" {
		return consts.ErrBadOrigin
	}
	group.Version = iam_repo.NewResourceVersion()
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	subj, err := s.membersFetcher.Fetch(group.Members)
	if err != nil {
		return err
	}
	group.Groups = subj.Groups
	group.ServiceAccounts = subj.ServiceAccounts
	group.Users = subj.Users

	return s.repo.Create(group)
}

func (s *GroupService) Update(group *model.Group) error {
	stored, err := s.repo.GetByID(group.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != s.tenantUUID {
		return consts.ErrNotFound
	}
	if stored.Origin != group.Origin {
		return consts.ErrBadOrigin
	}
	if stored.Version != group.Version {
		return consts.ErrBadVersion
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}

	tenant, err := s.tenantsRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}
	// Update
	group.TenantUUID = s.tenantUUID
	group.Version = iam_repo.NewResourceVersion()
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	subj, err := s.membersFetcher.Fetch(group.Members)
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

func (s *GroupService) Delete(origin consts.ObjectOrigin, id model.GroupUUID) error {
	group, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if group.Origin != origin {
		return consts.ErrBadOrigin
	}
	if group.Archived() {
		return consts.ErrIsArchived
	}
	err = s.repo.CleanChildrenSliceIndexes(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(id, memdb.NewArchiveMark())
}

func (s *GroupService) SetExtension(ext *model.Extension) error {
	stored, err := s.repo.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	if stored.Extensions == nil {
		stored.Extensions = make(map[consts.ObjectOrigin]*model.Extension)
	}
	stored.Extensions[ext.Origin] = ext
	return s.repo.Update(stored)
}

func (s *GroupService) UnsetExtension(origin consts.ObjectOrigin, uuid model.GroupUUID) error {
	stored, err := s.repo.GetByID(uuid)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}

	if stored.Extensions == nil {
		return nil
	}
	delete(stored.Extensions, origin)
	return s.repo.Update(stored)
}

func (s *GroupService) List(tid model.TenantUUID, showArchived bool) ([]*model.Group, error) {
	return s.repo.List(tid, showArchived)
}
