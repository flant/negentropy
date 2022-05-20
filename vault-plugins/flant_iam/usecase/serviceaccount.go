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

type ServiceAccountService struct {
	tenantUUID model.TenantUUID
	origin     consts.ObjectOrigin

	tenantRepo          *iam_repo.TenantRepository
	repo                *iam_repo.ServiceAccountRepository
	identitySharingRepo *iam_repo.IdentitySharingRepository
	groupRepo           *iam_repo.GroupRepository
}

func ServiceAccounts(db *io.MemoryStoreTxn, origin consts.ObjectOrigin, tid model.TenantUUID) *ServiceAccountService {
	return &ServiceAccountService{
		tenantUUID: tid,
		origin:     origin,

		repo:                iam_repo.NewServiceAccountRepository(db),
		tenantRepo:          iam_repo.NewTenantRepository(db),
		identitySharingRepo: iam_repo.NewIdentitySharingRepository(db),
		groupRepo:           iam_repo.NewGroupRepository(db),
	}
}

func (s *ServiceAccountService) Create(sa *model.ServiceAccount) error {
	tenant, err := s.tenantRepo.GetByID(sa.TenantUUID)
	if err != nil {
		return err
	}
	_, err = s.repo.GetByIdentifierAtTenant(sa.TenantUUID, sa.Identifier)
	if err != nil && !errors.Is(err, consts.ErrNotFound) {
		return err
	}
	if err == nil {
		return fmt.Errorf("%w: identifier:%s at tenant:%s", consts.ErrAlreadyExists, sa.Identifier, sa.TenantUUID)
	}
	if sa.Version != "" {
		return consts.ErrBadVersion
	}
	if sa.Origin == "" {
		return consts.ErrBadOrigin
	}
	sa.Version = iam_repo.NewResourceVersion()
	sa.FullIdentifier = iam_repo.CalcServiceAccountFullIdentifier(sa.Identifier, tenant.Identifier)

	return s.repo.Create(sa)
}

func (s *ServiceAccountService) GetByID(id model.ServiceAccountUUID) (*model.ServiceAccount, error) {
	return s.repo.GetByID(id)
}

/*
TODO
	* Из-за того, что в очереди формата TokenGenerationNumber стоит ttl 30 дней – token_ttl не может быть больше 30 дней.
		См. подробнее следующий пункт и описание формата очереди.

TODO Логика создания/обновления сервис аккаунта:
	* type object_with_resource_version
	* type tenanted_object
	* validate_tenant(запрос, объект из стора)
	* validate_resource_version(запрос, entry)
	* пробуем загрузить объект, если объект есть, то:
	* валидируем resource_version
	* валидируем тенанта
	* валидируем builtin_type_name
	* если объекта нет, то:
	* валидируем, что нам не передан resource_version
*/
func (s *ServiceAccountService) Update(sa *model.ServiceAccount) error {
	stored, err := s.repo.GetByID(sa.UUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	// Validate
	if stored.TenantUUID != s.tenantUUID {
		return consts.ErrNotFound
	}
	if stored.Origin != sa.Origin {
		return consts.ErrBadOrigin
	}
	if stored.Version != sa.Version {
		return consts.ErrBadVersion
	}

	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	// Update
	sa.TenantUUID = s.tenantUUID
	sa.Version = iam_repo.NewResourceVersion()
	sa.FullIdentifier = iam_repo.CalcServiceAccountFullIdentifier(sa.Identifier, tenant.Identifier)

	// Preserve fields, that are not always accessible from the outside, e.g. from HTTP API
	if sa.Extensions == nil {
		sa.Extensions = stored.Extensions
	}

	return s.repo.Update(sa)
}

/*
TODO
	* При удалении необходимо удалить все “вложенные” объекты (Token и ServiceAccountPassword).
	* При удалении необходимо удалить из всех связей (из групп, из role_binding’ов, из approval’ов и пр.)
*/

func (s *ServiceAccountService) Delete(id model.ServiceAccountUUID) error {
	sa, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if sa.Origin != s.origin {
		return consts.ErrBadOrigin
	}

	err = s.repo.CleanChildrenSliceIndexes(id)
	if err != nil {
		return err
	}
	return s.repo.CascadeDelete(id, memdb.NewArchiveMark())
}

func (s *ServiceAccountService) List(showShared bool, showArchived bool) ([]*model.ServiceAccount, error) {
	if showShared {
		sharedGroupUUIDs := map[model.GroupUUID]struct{}{}
		iss, err := s.identitySharingRepo.ListForDestinationTenant(s.tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("collecting identity_sharings:%w", err)
		}
		for _, is := range iss {
			for _, g := range is.Groups {
				gs, err := s.groupRepo.FindAllChildGroups(g, showArchived)
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

		sharedSAsUUIDs := map[model.UserUUID]struct{}{}
		for gUUID := range sharedGroupUUIDs {
			g, err := s.groupRepo.GetByID(gUUID)
			if err != nil {
				return nil, fmt.Errorf("collecting service_accounts of shared groups:%w", err)
			}
			for _, saUUID := range g.ServiceAccounts {
				sharedSAsUUIDs[saUUID] = struct{}{}
			}
		}
		serviceAccounts, err := s.repo.List(s.tenantUUID, showArchived)
		if err != nil {
			return nil, fmt.Errorf("collecting own service_accounts:%w", err)
		}
		// remove "self" sa from shared
		for _, s := range serviceAccounts {
			delete(sharedSAsUUIDs, s.UUID)
		}
		for sharedSaUUID := range sharedSAsUUIDs {
			sharedServiceAccount, err := s.repo.GetByID(sharedSaUUID)
			if err != nil {
				return nil, fmt.Errorf("getting shared service_account:%w", err)
			}
			serviceAccounts = append(serviceAccounts, sharedServiceAccount)
		}
		return serviceAccounts, nil
	}

	return s.repo.List(s.tenantUUID, showArchived)
}

func (s *ServiceAccountService) SetExtension(ext *model.Extension) error {
	obj, err := s.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[consts.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return s.Update(obj)
}

func (s *ServiceAccountService) UnsetExtension(origin consts.ObjectOrigin, uuid string) error {
	obj, err := s.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return s.Update(obj)
}

// func (s *ServiceAccountService) CascadeRestore(id model.ServiceAccountUUID) (*model.ServiceAccount, error) {
//	return s.repo.CascadeRestore(id)
// }
