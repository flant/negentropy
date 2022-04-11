package usecase

import (
	"errors"
	"fmt"

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
	db                  *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantUUID          model.TenantUUID
	origin              consts.ObjectOrigin
	repo                *iam_repo.GroupRepository
	identitySharingRepo *iam_repo.IdentitySharingRepository
	tenantsRepo         *iam_repo.TenantRepository
	membersFetcher      *MembersFetcher
}

func Groups(db *io.MemoryStoreTxn, tid model.TenantUUID, origin consts.ObjectOrigin) *GroupService {
	return &GroupService{
		db:                  db,
		tenantUUID:          tid,
		origin:              origin,
		repo:                iam_repo.NewGroupRepository(db),
		identitySharingRepo: iam_repo.NewIdentitySharingRepository(db),
		tenantsRepo:         iam_repo.NewTenantRepository(db),
		membersFetcher:      NewMembersFetcher(db),
	}
}

func (s *GroupService) Create(group *model.Group) error {
	tenant, err := s.tenantsRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}
	_, err = s.repo.GetByIdentifierAtTenant(group.TenantUUID, group.Identifier)
	if err != nil && !errors.Is(err, consts.ErrNotFound) {
		return err
	}
	if err == nil {
		return fmt.Errorf("%w: identifier:%s at tenant:%s", consts.ErrAlreadyExists, group.Identifier, group.TenantUUID)
	}
	if group.Version != "" {
		return consts.ErrBadVersion
	}
	group.Origin = s.origin
	group.Version = iam_repo.NewResourceVersion()
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	subj, err := s.membersFetcher.Fetch(group.Members)
	if err != nil {
		return err
	}
	err = checkMembersOwnedToTenant(s.db, *subj, group.TenantUUID)
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
	if stored.Origin != s.origin {
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
	group.Origin = s.origin
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	subj, err := s.membersFetcher.Fetch(group.Members)
	if err != nil {
		return err
	}
	err = checkMembersOwnedToTenant(s.db, *subj, group.TenantUUID)
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

func (s *GroupService) Delete(id model.GroupUUID) error {
	group, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if group.Origin != s.origin {
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

func (s *GroupService) List(showShared bool, showArchived bool) ([]*model.Group, error) {
	if showShared {
		sharedGroupUUIDs := map[model.GroupUUID]struct{}{}
		iss, err := s.identitySharingRepo.ListForDestinationTenant(s.tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("collecting identity_sharings:%w", err)
		}
		for _, is := range iss {
			for _, g := range is.Groups {
				gs, err := s.repo.FindAllChildGroups(g, showArchived)
				if err != nil {
					return nil, fmt.Errorf("collecting shared groups:%w", err)
				}
				for candidate := range gs {
					if _, alreadyCollected := sharedGroupUUIDs[candidate]; !alreadyCollected {
						sharedGroupUUIDs[candidate] = struct{}{}
					}
				}
			}
		}
		groups, err := s.repo.List(s.tenantUUID, showArchived)
		if err != nil {
			return nil, fmt.Errorf("collecting own groups:%w", err)
		}
		for sharedGroupUUID := range sharedGroupUUIDs {
			sharedGroup, err := s.repo.GetByID(sharedGroupUUID)
			if err != nil {
				return nil, fmt.Errorf("getting shared group:%w", err)
			}
			groups = append(groups, sharedGroup)
		}
		return groups, nil
	}
	return s.repo.List(s.tenantUUID, showArchived)
}

func (s *GroupService) GetByID(id model.GroupUUID) (*model.Group, error) {
	return s.repo.GetByID(id)
}

func (s *GroupService) RemoveUsersFromGroup(groupUUID model.GroupUUID, userUUIDs ...model.UserUUID) error {
	group, err := s.GetByID(groupUUID)
	if err != nil {
		return err
	}
	for _, userUUID := range userUUIDs {
		targetUserIDX := -1
		for i := range group.Users {
			if group.Users[i] == userUUID {
				targetUserIDX = i
				break
			}
		}
		if targetUserIDX > -1 {
			group.Users = append(group.Users[:targetUserIDX], group.Users[targetUserIDX+1:]...) // nolint:gocritic
			group.FixMembers()
			err = s.Update(group)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *GroupService) AddUsersToGroup(groupUUID model.GroupUUID, userUUIDs ...model.UserUUID) error {
	group, err := s.GetByID(groupUUID)
	if err != nil {
		return err
	}
	for _, extraUserUUID := range userUUIDs {
		group.Users = append(group.Users, extraUserUUID)
		group.Members = append(group.Members, model.MemberNotation{
			Type: model.UserType,
			UUID: extraUserUUID,
		})
	}
	return s.Update(group)
}
