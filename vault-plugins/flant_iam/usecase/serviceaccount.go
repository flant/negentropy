package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ServiceAccountService struct {
	tenantUUID model.TenantUUID
	origin     model.ObjectOrigin

	repo       *model.ServiceAccountRepository
	tenantRepo *model.TenantRepository

	childrenDeleters []DeleterByParent
}

func ServiceAccounts(db *io.MemoryStoreTxn, origin model.ObjectOrigin, tid model.TenantUUID) *ServiceAccountService {
	return &ServiceAccountService{
		tenantUUID: tid,
		origin:     origin,

		repo:       model.NewServiceAccountRepository(db),
		tenantRepo: model.NewTenantRepository(db),

		childrenDeleters: []DeleterByParent{
			MultipassDeleter(db),
			PasswordDeleter(db),
		},
	}
}

func (s *ServiceAccountService) Create(sa *model.ServiceAccount) error {
	tenant, err := s.tenantRepo.GetByID(sa.TenantUUID)
	if err != nil {
		return err
	}
	if sa.Version != "" {
		return model.ErrBadVersion
	}
	if sa.Origin == "" {
		return model.ErrBadOrigin
	}
	sa.Version = model.NewResourceVersion()
	sa.FullIdentifier = model.CalcServiceAccountFullIdentifier(sa.Identifier, tenant.Identifier)

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

	// Validate
	if stored.TenantUUID != s.tenantUUID {
		return model.ErrNotFound
	}
	if stored.Origin != sa.Origin {
		return model.ErrBadOrigin
	}
	if stored.Version != sa.Version {
		return model.ErrBadVersion
	}

	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	// Update
	sa.TenantUUID = s.tenantUUID
	sa.Version = model.NewResourceVersion()
	sa.FullIdentifier = model.CalcServiceAccountFullIdentifier(sa.Identifier, tenant.Identifier)

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
		return model.ErrBadOrigin
	}
	archivingTimestamp, archivingHash := ArchivingLabel()
	if err := deleteChildren(id, s.childrenDeleters, archivingTimestamp, archivingHash); err != nil {
		return err
	}
	return s.repo.Delete(id, archivingTimestamp, archivingHash)
}

func (s *ServiceAccountService) List(showArchived bool) ([]*model.ServiceAccount, error) {
	return s.repo.List(s.tenantUUID, showArchived)
}

func (s *ServiceAccountService) SetExtension(ext *model.Extension) error {
	obj, err := s.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[model.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return s.Update(obj)
}

func (s *ServiceAccountService) UnsetExtension(origin model.ObjectOrigin, uuid string) error {
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
