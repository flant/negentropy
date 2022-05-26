package usecase

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

// iam_usecase.Tenants implements every thing, except identity-sharing for flant-all
type ClientService struct {
	iam_usecase.TenantService
	identitySharingRepo   *iam_repo.IdentitySharingRepository
	roleBindingRepository *iam_repo.RoleBindingRepository
	userRepo              *iam_repo.UserRepository
	groupRepo             *iam_repo.GroupRepository
	liveConfig            *config.FlantFlowConfig
}

func Clients(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) *ClientService {
	return &ClientService{
		TenantService:         *iam_usecase.Tenants(db, consts.OriginFlantFlow),
		identitySharingRepo:   iam_repo.NewIdentitySharingRepository(db),
		roleBindingRepository: iam_repo.NewRoleBindingRepository(db),
		userRepo:              iam_repo.NewUserRepository(db),
		groupRepo:             iam_repo.NewGroupRepository(db),
		liveConfig:            liveConfig,
	}
}

func (s *ClientService) List(showArchived bool) ([]model.Tenant, error) {
	tenants, err := s.TenantService.List(showArchived)
	if err != nil {
		return nil, err
	}
	clients := make([]model.Tenant, 0, len(tenants))
	for _, t := range tenants {
		clients = append(clients, makeClient(t))
	}
	return clients, nil
}

func makeClient(t *model.Tenant) model.Tenant {
	var result model.Tenant = *t
	result.Origin = ""
	return result
}

func (s *ClientService) Create(t *model.Tenant, primaryClientAdministrators []model.UserUUID) (*model.Tenant, error) {
	if len(primaryClientAdministrators) == 0 {
		return nil, fmt.Errorf("%w: empty primary_administrators", consts.ErrInvalidArg)
	}
	err := s.TenantService.Create(t)
	if err != nil {
		return nil, err
	}
	result := makeClient(t)
	if s.liveConfig.FlantTenantUUID == "" {
		return &result, nil
	}
	is := &model.IdentitySharing{
		UUID:                  uuid.New(),
		SourceTenantUUID:      s.liveConfig.FlantTenantUUID,
		DestinationTenantUUID: t.UUID,
		Version:               uuid.New(),
		Origin:                consts.OriginFlantFlow,
		Groups:                []model.GroupUUID{s.liveConfig.AllFlantGroupUUID},
	}
	err = s.identitySharingRepo.Create(is)
	if err != nil {
		return &result, err
	}
	return &result, s.createPrimaryAdministrators(t, primaryClientAdministrators)
}

func (s *ClientService) GetByID(id model.TenantUUID) (*model.Tenant, error) {
	t, err := s.TenantService.GetByID(id)
	if err != nil {
		return nil, err
	}
	result := makeClient(t)
	result.Origin = ""
	return &result, nil
}

func (s *ClientService) Update(updated *model.Tenant) (*model.Tenant, error) {
	err := s.TenantService.Update(updated)
	if err != nil {
		return nil, err
	}
	result := makeClient(updated)
	result.Origin = ""
	return &result, nil
}

func (s *ClientService) Restore(id model.TenantUUID, fullRestore bool) (*model.Tenant, error) {
	t, err := s.TenantService.Restore(id, fullRestore)
	if err != nil {
		return nil, err
	}
	iss, err := s.identitySharingRepo.ListForDestinationTenant(id)
	for _, is := range iss {
		if is.SourceTenantUUID == s.liveConfig.FlantTenantUUID && is.Origin == consts.OriginFlantFlow &&
			len(is.Groups) == 1 && is.Groups[0] == s.liveConfig.AllFlantGroupUUID {
			if is.Archived() {
				is.Restore()
				err = s.identitySharingRepo.Update(is)
				if err != nil {
					return nil, err
				}
			}
			break
		}
	}
	result := makeClient(t)
	result.Origin = ""
	return &result, nil
}

func (s *ClientService) createPrimaryAdministrators(t *model.Tenant, administrators []model.UserUUID) error {
	// collect users
	usersByTenant := map[model.TenantUUID][]model.UserUUID{}
	for _, adminUUID := range administrators {
		user, err := s.userRepo.GetByID(adminUUID)
		if err != nil {
			return fmt.Errorf("client primary admin uuid=%s :%w", adminUUID, err)
		}
		usersByTenant[user.TenantUUID] = append(usersByTenant[user.TenantUUID], adminUUID)
	}
	// create groups & identity sharings
	for tenantUUID, userUUIDs := range usersByTenant {
		tenant, err := s.TenantService.GetByID(tenantUUID)
		if err != nil {
			return err
		}
		members := buildMembers(model.UserType, userUUIDs)
		group := &model.Group{
			UUID:       uuid.New(),
			TenantUUID: tenant.UUID,
			Version:    uuid.New(),
			Identifier: fmt.Sprintf("shared_to_%s", t.Identifier),
			Users:      userUUIDs,
			Members:    members,
			Origin:     consts.OriginIAM,
		}
		group.FullIdentifier = iam_usecase.CalcGroupFullIdentifier(group, tenant)
		err = s.groupRepo.Create(group)
		if err != nil {
			return err
		}
		err = s.identitySharingRepo.Create(&model.IdentitySharing{
			UUID:                  uuid.New(),
			SourceTenantUUID:      tenant.UUID,
			DestinationTenantUUID: t.UUID,
			Version:               uuid.New(),
			Origin:                consts.OriginIAM,
			Groups:                []model.GroupUUID{group.UUID},
		})
		if err != nil {
			return err
		}
	}
	roles := []model.BoundRole{}
	for _, r := range s.liveConfig.ClientPrimaryAdministratorsRoles {
		roles = append(roles, model.BoundRole{Name: r})
	}

	// create RB
	return s.roleBindingRepository.Create(&model.RoleBinding{
		ArchiveMark: memdb.ArchiveMark{},
		UUID:        uuid.New(),
		TenantUUID:  t.UUID,
		Version:     uuid.New(),
		Description: "autocreated rolebinding for primary administrators",
		Users:       administrators,
		Members:     buildMembers(model.UserType, administrators),
		AnyProject:  true,
		Roles:       roles,
		Origin:      consts.OriginIAM,
	})
}
