package usecase

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type RoleBindingService struct {
	db              *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	repo            *iam_repo.RoleBindingRepository
	tenantsRepo     *iam_repo.TenantRepository
	roleRepoository *iam_repo.RoleRepository

	memberFetcher        *MembersFetcher
	memberNotationMapper MemberNotationMapper
	projectMapper        ProjectMapper
}

func RoleBindings(db *io.MemoryStoreTxn) *RoleBindingService {
	return &RoleBindingService{
		db:              db,
		repo:            iam_repo.NewRoleBindingRepository(db),
		memberFetcher:   NewMembersFetcher(db),
		tenantsRepo:     iam_repo.NewTenantRepository(db),
		roleRepoository: iam_repo.NewRoleRepository(db),

		memberNotationMapper: NewMemberNotationMapper(db),
		projectMapper:        NewProjectMapper(db),
	}
}

func (s *RoleBindingService) Create(rb *model.RoleBinding) (*DenormalizedRoleBinding, error) {
	// Validate
	if rb.Origin == "" {
		return nil, consts.ErrBadOrigin
	}
	if rb.Version != "" {
		return nil, consts.ErrBadVersion
	}
	rb.Version = iam_repo.NewResourceVersion()

	// Refill data
	subj, err := s.memberFetcher.Fetch(rb.Members)
	if err != nil {
		return nil, fmt.Errorf("RoleBindingService.Create:%w", err)
	}
	if err := s.checkRoles(rb.Roles); err != nil {
		return nil, err
	}
	// TODO check - owned or shared
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users
	if rb.UUID == "" {
		rb.UUID = uuid.New()
	}
	err = s.repo.Create(rb)
	if err != nil {
		return nil, fmt.Errorf("RoleBindingService.Create:%s", err)
	}
	return s.denormalizeRoleBinding(rb)
}

func (s *RoleBindingService) Update(rb *model.RoleBinding) (*DenormalizedRoleBinding, error) {
	// Validate
	stored, err := s.repo.GetByID(rb.UUID)
	if err != nil {
		return nil, err
	}
	if stored.Archived() {
		return nil, consts.ErrIsArchived
	}
	if rb.Origin != stored.Origin {
		return nil, consts.ErrBadOrigin
	}
	if stored.Version != rb.Version {
		return nil, consts.ErrBadVersion
	}
	rb.Version = iam_repo.NewResourceVersion()
	if stored.TenantUUID != rb.TenantUUID {
		return nil, consts.ErrNotFound
	}
	// Refill data
	subj, err := s.memberFetcher.Fetch(rb.Members)
	if err != nil {
		return nil, err
	}
	if err := s.checkRoles(rb.Roles); err != nil {
		return nil, err
	}
	// TODO check - owned or shared
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users

	// Preserve fields, that are not always accessible from the outside, e.g. from HTTP API
	if rb.Extensions == nil {
		rb.Extensions = stored.Extensions
	}

	// Store
	err = s.repo.Update(rb)
	if err != nil {
		return nil, fmt.Errorf("RoleBindingService.Update:%s", err)
	}
	return s.denormalizeRoleBinding(rb)
}

func (s *RoleBindingService) Delete(origin consts.ObjectOrigin, id model.RoleBindingUUID) error {
	roleBinding, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if roleBinding.Origin != origin {
		return consts.ErrBadOrigin
	}
	return s.repo.CascadeDelete(id, memdb.NewArchiveMark())
}

func (s *RoleBindingService) SetExtension(ext *model.Extension) (*DenormalizedRoleBinding, error) {
	obj, err := s.repo.GetByID(ext.OwnerUUID)
	if err != nil {
		return nil, err
	}
	if obj.Archived() {
		return nil, consts.ErrIsArchived
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[consts.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	err = s.repo.Update(obj)
	if err != nil {
		return nil, fmt.Errorf("RoleBindingService.Update:%s", err)
	}
	return s.denormalizeRoleBinding(obj)
}

func (s *RoleBindingService) UnsetExtension(origin consts.ObjectOrigin, id model.RoleBindingUUID) (*DenormalizedRoleBinding, error) {
	obj, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if obj.Archived() {
		return nil, consts.ErrIsArchived
	}
	if obj.Extensions == nil {
		return s.denormalizeRoleBinding(obj)
	}
	delete(obj.Extensions, origin)
	err = s.repo.Update(obj)
	if err != nil {
		return nil, fmt.Errorf("RoleBindingService.Update:%s", err)
	}
	return s.denormalizeRoleBinding(obj)
}

func (s *RoleBindingService) List(tid model.TenantUUID, showArchived bool) ([]*DenormalizedRoleBinding, error) {
	rbs, err := s.repo.List(tid, showArchived)
	if err != nil {
		return nil, err
	}
	return s.denormalizeRoleBindings(rbs)
}

func (s *RoleBindingService) GetByID(id model.RoleBindingUUID) (*DenormalizedRoleBinding, error) {
	rb, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return s.denormalizeRoleBinding(rb)
}

type DenormalizedRoleBinding struct {
	memdb.ArchiveMark

	UUID       model.RoleBindingUUID `json:"uuid"` // PK
	TenantUUID model.TenantUUID      `json:"tenant_uuid"`
	Version    string                `json:"resource_version"`

	Description string `json:"description"`

	ValidTill  int64 `json:"valid_till"` // if ==0 => valid forever
	RequireMFA bool  `json:"require_mfa"`

	Members []DenormalizedMemberNotation `json:"members"`

	AnyProject bool                        `json:"any_project"`
	Projects   []ProjectUUIDWithIdentifier `json:"projects"`

	Roles []model.BoundRole `json:"roles"`

	Origin consts.ObjectOrigin `json:"origin"`
}

func (s *RoleBindingService) denormalizeRoleBindings(rbs []*model.RoleBinding) ([]*DenormalizedRoleBinding, error) {
	result := make([]*DenormalizedRoleBinding, 0, len(rbs))
	for _, rb := range rbs {
		drb, err := s.denormalizeRoleBinding(rb)
		if err != nil {
			return nil, err
		}
		result = append(result, drb)
	}
	return result, nil
}

func (s *RoleBindingService) denormalizeRoleBinding(rb *model.RoleBinding) (*DenormalizedRoleBinding, error) {
	denormilizedMembers, err := s.memberNotationMapper.Denormalize(rb.Members)
	if err != nil {
		return nil, err
	}
	denormilizedProjects, err := s.projectMapper.Denormalize(rb.Projects)
	if err != nil {
		return nil, err
	}
	return &DenormalizedRoleBinding{
		ArchiveMark: rb.ArchiveMark,
		UUID:        rb.UUID,
		TenantUUID:  rb.TenantUUID,
		Version:     rb.Version,
		Description: rb.Description,
		ValidTill:   rb.ValidTill,
		RequireMFA:  rb.RequireMFA,
		Members:     denormilizedMembers,
		AnyProject:  rb.AnyProject,
		Projects:    denormilizedProjects,
		Roles:       rb.Roles,
		Origin:      rb.Origin,
	}, nil
}

func (s *RoleBindingService) checkRoles(roles []model.BoundRole) error {
	for _, br := range roles {
		role, err := s.roleRepoository.GetByID(br.Name)
		if err != nil {
			return fmt.Errorf("%s:%w", br.Name, err)
		}
		if role.ForbinddenDirectUse {
			return fmt.Errorf("%w:%s - prohibited direct use in rolebinding", consts.ErrInvalidArg, br.Name)
		}
		if err = checkOptions(role.OptionsSchema, br.Options); err != nil {
			return fmt.Errorf("%w: check options for role %q: %s", consts.ErrInvalidArg, br.Name, err.Error())
		}
	}
	return nil
}
