package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

// iam_usecase.Tenants implements every thing, except identity-sharing for flant-all
type ClientService struct {
	iam_usecase.TenantService
	identitySharingRepo *iam_repo.IdentitySharingRepository
	liveConfig          *config.FlantFlowConfig
}

func Clients(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) *ClientService {
	return &ClientService{
		TenantService:       *iam_usecase.Tenants(db, consts.OriginFlantFlow),
		identitySharingRepo: iam_repo.NewIdentitySharingRepository(db),
		liveConfig:          liveConfig,
	}
}

func (s *ClientService) Create(t *model.Tenant) error {
	err := s.TenantService.Create(t)
	if err != nil {
		return err
	}
	is := &model.IdentitySharing{
		UUID:                  uuid.New(),
		SourceTenantUUID:      s.liveConfig.FlantTenantUUID,
		DestinationTenantUUID: t.UUID,
		Version:               uuid.New(),
		Groups:                []model.GroupUUID{s.liveConfig.AllFlantGroup},
	}
	return s.identitySharingRepo.Create(is)
}

func (s *ClientService) Restore(id model.TenantUUID, fullRestore bool) (*model.Tenant, error) {
	t, err := s.TenantService.Restore(id, fullRestore)
	if err != nil {
		return nil, err
	}
	iss, err := s.identitySharingRepo.ListForDestinationTenant(id)
	for _, is := range iss {
		if is.SourceTenantUUID == s.liveConfig.FlantTenantUUID &&
			len(is.Groups) == 1 && is.Groups[0] == s.liveConfig.AllFlantGroup {
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
	return t, err
}
