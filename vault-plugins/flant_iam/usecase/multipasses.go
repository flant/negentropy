package usecase

import (
	"fmt"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	jwttoken "github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

/*

Особенности:
	* Не работает, если не выполнен /jwt/enable.
	* Это JWT которые используются в auth.negentropy.flant.com для получения JWT access токенов, которые уже дают доступ непосредственно к сервисам. По-сути это refresh токены.
	* При создании нового токена генерируется (и сохраняются), но пользователю не отображается, длинный случайный идентификатор (соль).
	* Обычно при каждом выпуске инкрементируется номер поколения токена, и новое значение публикуется в специальную общую очередь в Kafka (см. подробнее формат очереди TokenGenerationNumber). Но так как при создании токена номер поколения ноль – ничего в очередь не публикуется (при отсутствии в этой очереди данных для токена считается, что номер его поколения 0).
	* При создании на основании переданного token_max_ttl создается параметр valid_till с конкретным временем окончания токена.
	* Пользователю выдается “первичный” JWT токен, содержащий:
		iss: issuer из /jwt/configure (в нашем случае это https://auth.negentropy.flant.com)
		aud: own_audience из /jwt/configure (в нашем случае это auth.negentropy.flant.com)
		sub: <token_uuid>
		jti: контрольная сумма от соли и номера поколения
	* При этом плагин flant_iam_auth:
		проверяет корректность jti при аутентификации по токену (вход по устаревшему jti невозможен, но ввиду
		распределенных свойств системы есть небольшое окно, в которое возможно использование родированного токена);
		предоставляет метод, позволяющий провести ротацию этого токена (с инкрементацией поколения).

description – комментарий о том, где это используется и зачем (чтобы потом можно было вспомнить).
...
allowed_roles – список ролей, которые может использовать этот токен (итоговый список вычисляется на основании пересечения role_binding’ов и этого массива, можно использовать *, например: “iam.*”)
…
allowed_cidrs
token_ttl – период жизни JWT токена (по-умолчанию 2 недели, токен должен быть ротирован не реже, чем раз в TTL);
	Важно! Не может быть больше чем время rotation_period у JWKS, и не может быть больше, чем время хранения сообщений в очереди TokenGenerationNumber.
token_max_ttl – максимальная продолжительность жизни токенов (по-умолчанию 0, после окончания этого TTL токен невозможножно больше ротировать, он автоматически удаляется);


*/

type MultipassService struct {
	// dependencies
	repo       *iam_repo.MultipassRepository
	tenantRepo *iam_repo.TenantRepository
	userRepo   *iam_repo.UserRepository
	saRepo     *iam_repo.ServiceAccountRepository

	// context
	origin     consts.ObjectOrigin
	ownerType  model.MultipassOwnerType
	tenantUUID model.TenantUUID
	ownerUUID  model.OwnerUUID
}

func UserMultipasses(db *io.MemoryStoreTxn, origin consts.ObjectOrigin, tid model.TenantUUID, uid model.OwnerUUID) *MultipassService {
	return Multipasses(db, origin, model.MultipassOwnerUser, tid, uid)
}

func ServiceAccountMultipasses(db *io.MemoryStoreTxn, origin consts.ObjectOrigin, tid model.TenantUUID, said model.OwnerUUID) *MultipassService {
	return Multipasses(db, origin, model.MultipassOwnerServiceAccount, tid, said)
}

func Multipasses(db *io.MemoryStoreTxn, origin consts.ObjectOrigin, otype model.MultipassOwnerType, tid model.TenantUUID, oid model.OwnerUUID) *MultipassService {
	return &MultipassService{
		repo:       iam_repo.NewMultipassRepository(db),
		tenantRepo: iam_repo.NewTenantRepository(db),
		userRepo:   iam_repo.NewUserRepository(db),
		saRepo:     iam_repo.NewServiceAccountRepository(db),

		origin:     origin,
		ownerType:  otype,
		tenantUUID: tid,
		ownerUUID:  oid,
	}
}

func (r *MultipassService) Create(ttl, maxTTL time.Duration, cidrs, roles []string, description string) (*model.Multipass, error) {
	err := r.validateContext()
	if err != nil {
		return nil, err
	}

	mp := &model.Multipass{
		TenantUUID: r.tenantUUID,
		OwnerUUID:  r.ownerUUID,
		OwnerType:  r.ownerType,
		Origin:     r.origin,

		Description: description,
		TTL:         ttl,    // TODO validate TTL
		MaxTTL:      maxTTL, // TODO validate MaxTTL
		CIDRs:       cidrs,  // TODO validate CIDRs
		Roles:       roles,  // TODO validate Roles

		UUID:      uuid.New(),
		ValidTill: time.Now().Add(ttl).Unix(),
		Salt:      uuid.New(),
	}

	err = r.repo.Create(mp)
	if err != nil {
		return nil, err
	}
	return mp, nil
}

// CreateWithJWT saves a Multipass object and generate jwt.
func (r *MultipassService) CreateWithJWT(
	issueMultipass jwt.MultipassIssFn, // jwt
	ttl, maxTTL time.Duration, cidrs, roles []string, description string, // multipass
) (string, *model.Multipass, error) {
	mp, err := r.Create(ttl, maxTTL, cidrs, roles, description)
	if err != nil {
		return "", nil, err
	}

	// Generate JWT
	options := &jwttoken.PrimaryTokenOptions{
		TTL:  mp.TTL,
		UUID: mp.UUID,
		JTI: jwttoken.TokenJTI{
			Generation: 0,
			SecretSalt: mp.Salt,
		},
	}

	jwtString, err := issueMultipass(options)
	if err != nil {
		return "", nil, err
	}

	safeMp := iam_repo.OmitSensitive(mp).(model.Multipass)
	return jwtString, &safeMp, nil
}

func (r *MultipassService) Delete(id model.MultipassUUID) error {
	err := r.validateContext()
	if err != nil {
		return err
	}
	archivingTimestamp, archivingHash := ArchivingLabel()

	return r.CascadeDelete(id, archivingTimestamp, archivingHash)
}

func (r *MultipassService) GetByID(id model.MultipassUUID) (*model.Multipass, error) {
	err := r.validateContext()
	if err != nil {
		return nil, err
	}

	mp, err := r.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if mp.OwnerType != r.ownerType {
		return nil, consts.ErrNotFound
	}
	safeMp := iam_repo.OmitSensitive(mp).(model.Multipass)
	return &safeMp, nil
}

// TODO add listing by origin
func (r *MultipassService) List(showArchived bool) ([]*model.Multipass, error) {
	err := r.validateContext()
	if err != nil {
		return nil, err
	}
	return r.repo.List(r.ownerUUID, showArchived)
}

func (r *MultipassService) PublicList(showArchived bool) ([]*model.Multipass, error) {
	mps, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}

	safeMps := make([]*model.Multipass, len(mps))
	for i := range mps {
		safe := iam_repo.OmitSensitive(mps[i]).(model.Multipass)
		safeMps[i] = &safe
	}

	return safeMps, nil
}

func (r *MultipassService) validateContext() error {
	if err := consts.ValidateOrigin(r.origin); err != nil {
		return err
	}
	_, err := r.tenantRepo.GetByID(r.tenantUUID)
	if err != nil {
		return err
	}

	if r.ownerType == model.MultipassOwnerUser {
		owner, err := r.userRepo.GetByID(r.ownerUUID)
		if err != nil {
			return err
		}
		if owner.TenantUUID != r.tenantUUID {
			return consts.ErrNotFound
		}
	}

	if r.ownerType == model.MultipassOwnerServiceAccount {
		owner, err := r.saRepo.GetByID(r.ownerUUID)
		if err != nil {
			return err
		}
		if owner.TenantUUID != r.tenantUUID {
			return consts.ErrNotFound
		}
	}

	return nil
}

func (r *MultipassService) SetExtension(ext *model.Extension) error {
	if ext.OwnerType != model.ExtensionOwnerTypeServiceAccount && ext.OwnerType != model.ExtensionOwnerTypeUser {
		return fmt.Errorf("multipass extension is suppoted only for , got type %q", ext.OwnerType)
	}
	obj, err := r.repo.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[consts.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return r.repo.Update(obj)
}

func (r *MultipassService) UnsetExtension(origin consts.ObjectOrigin, uuid model.MultipassUUID) error {
	obj, err := r.repo.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return r.repo.Update(obj)
}

func (r *MultipassService) CascadeDelete(id model.MultipassUUID,
	archivingTimestamp model.UnixTime, archivingHash int64) error {
	err := r.validateContext()
	if err != nil {
		return err
	}

	return r.repo.Delete(id, archivingTimestamp, archivingHash)
}
