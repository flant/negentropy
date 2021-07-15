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
}

func ServiceAccounts(db *io.MemoryStoreTxn, origin model.ObjectOrigin, tid model.TenantUUID) *ServiceAccountService {
	return &ServiceAccountService{
		tenantUUID: tid,
		origin:     origin,

		repo:       model.NewServiceAccountRepository(db),
		tenantRepo: model.NewTenantRepository(db),
	}
}

func (r *ServiceAccountService) Create(sa *model.ServiceAccount) error {
	tenant, err := r.tenantRepo.GetByID(sa.TenantUUID)
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
	sa.FullIdentifier = CalcServiceAccountFullIdentifier(sa, tenant)

	return r.repo.Create(sa)
}

func (r *ServiceAccountService) GetByID(id model.ServiceAccountUUID) (*model.ServiceAccount, error) {
	return r.repo.GetByID(id)
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
func (r *ServiceAccountService) Update(sa *model.ServiceAccount) error {
	stored, err := r.repo.GetByID(sa.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != r.tenantUUID {
		return model.ErrNotFound
	}
	if stored.Origin != sa.Origin {
		return model.ErrBadOrigin
	}
	if stored.Version != sa.Version {
		return model.ErrBadVersion
	}

	tenant, err := r.tenantRepo.GetByID(r.tenantUUID)
	if err != nil {
		return err
	}

	// Update
	sa.TenantUUID = r.tenantUUID
	sa.Version = model.NewResourceVersion()
	sa.FullIdentifier = CalcServiceAccountFullIdentifier(sa, tenant)

	// Preserve fields, that are not always accessible from the outside, e.g. from HTTP API
	if sa.Extensions == nil {
		sa.Extensions = stored.Extensions
	}

	return r.repo.Update(sa)
}

/*
TODO
	* При удалении необходимо удалить все “вложенные” объекты (Token и ServiceAccountPassword).
	* При удалении необходимо удалить из всех связей (из групп, из role_binding’ов, из approval’ов и пр.)
*/

func (r *ServiceAccountService) Delete(id model.ServiceAccountUUID) error {
	sa, err := r.repo.GetByID(id)
	if err != nil {
		return err
	}
	if sa.Origin != r.origin {
		return model.ErrBadOrigin
	}
	return r.repo.Delete(id)
}

func (r *ServiceAccountService) List() ([]*model.ServiceAccount, error) {
	return r.repo.List(r.tenantUUID)
}

func (r *ServiceAccountService) SetExtension(ext *model.Extension) error {
	obj, err := r.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[model.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return r.Update(obj)
}

func (r *ServiceAccountService) UnsetExtension(origin model.ObjectOrigin, uuid string) error {
	obj, err := r.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return r.Update(obj)
}

// generic: <identifier>@serviceaccount.<tenant_identifier>
// builtin: <identifier>@<builtin_service_account_type>.serviceaccount.<tenant_identifier>
func CalcServiceAccountFullIdentifier(sa *model.ServiceAccount, tenant *model.Tenant) string {
	name := sa.Identifier
	domain := "serviceaccount." + tenant.Identifier
	if sa.BuiltinType != "" {
		domain = sa.BuiltinType + "." + domain
	}
	return name + "@" + domain
}
