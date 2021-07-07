package model

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	MultipassType  = "multipass" // also, memdb schema name
	OwnerForeignPK = "owner_uuid"
)

func MultipassSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			MultipassType: {
				Name: MultipassType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					OwnerForeignPK: {
						Name: OwnerForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "OwnerUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}

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

type MultipassOwnerType string

const (
	MultipassOwnerServiceAccount MultipassOwnerType = "service_account"
	MultipassOwnerUser           MultipassOwnerType = "user"
)

type Multipass struct {
	UUID        MultiPassUUID      `json:"uuid"` // PK
	TenantUUID  TenantUUID         `json:"tenant_uuid"`
	OwnerUUID   OwnerUUID          `json:"owner_uuid"`
	OwnerType   MultipassOwnerType `json:"owner_type"`
	Description string             `json:"description"`
	TTL         time.Duration      `json:"ttl"`
	MaxTTL      time.Duration      `json:"max_ttl"`
	ValidTill   int64              `json:"valid_till"`
	CIDRs       []string           `json:"allowed_cidrs"`
	Roles       []RoleName         `json:"allowed_roles" `
	Salt        string             `json:"salt,omitempty" sensitive:""`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"extensions"`
}

func (t *Multipass) ObjType() string {
	return MultipassType
}

func (t *Multipass) ObjId() MultiPassUUID {
	return t.UUID
}

func (t Multipass) Marshal(includeSensitive bool) ([]byte, error) {
	obj := t
	if !includeSensitive {
		t := OmitSensitive(t).(Multipass)
		obj = t
	}
	return jsonutil.EncodeJSON(obj)
}

func (t *Multipass) Unmarshal(data []byte) error {
	return jsonutil.DecodeJSON(data, t)
}

type MultipassRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewMultipassRepository(tx *io.MemoryStoreTxn) *MultipassRepository {
	return &MultipassRepository{db: tx}
}

func (r *MultipassRepository) validate(mp *Multipass) error {
	tenantRepo := NewTenantRepository(r.db)
	_, err := tenantRepo.GetByID(mp.TenantUUID)
	if err != nil {
		return err
	}

	if mp.OwnerType == MultipassOwnerUser {
		repo := NewUserRepository(r.db)
		owner, err := repo.GetByID(mp.OwnerUUID)
		if err != nil {
			return err
		}
		if owner.TenantUUID != mp.TenantUUID {
			return ErrNotFound
		}
	}

	if mp.OwnerType == MultipassOwnerServiceAccount {
		repo := NewServiceAccountRepository(r.db)
		owner, err := repo.GetByID(mp.OwnerUUID)
		if err != nil {
			return err
		}
		if owner.TenantUUID != mp.TenantUUID {
			return ErrNotFound
		}
	}

	return nil
}

func (r *MultipassRepository) save(mp *Multipass) error {
	return r.db.Insert(MultipassType, mp)
}

func (r *MultipassRepository) delete(filter *Multipass) error {
	return r.db.Delete(MultipassType, filter)
}

func (r *MultipassRepository) Create(mp *Multipass) error {
	if mp.Origin == "" {
		return ErrBadOrigin
	}
	err := r.validate(mp)
	if err != nil {
		return err
	}
	return r.save(mp)
}

func (r *MultipassRepository) Delete(filter *Multipass) error {
	if filter.Origin == "" {
		return ErrBadOrigin
	}
	err := r.validate(filter)
	if err != nil {
		return err
	}
	return r.delete(filter)
}

func (r *MultipassRepository) Get(filter *Multipass) (*Multipass, error) {
	err := r.validate(filter)
	if err != nil {
		return nil, err
	}
	return r.GetByID(filter.UUID)
}

func (r *MultipassRepository) GetByID(id MultiPassUUID) (*Multipass, error) {
	raw, err := r.db.First(MultipassType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	multipass := raw.(*Multipass)
	return multipass, nil
}

func (r *MultipassRepository) List(filter *Multipass) ([]MultiPassUUID, error) {
	err := r.validate(filter)
	if err != nil {
		return nil, err
	}

	iter, err := r.db.Get(MultipassType, OwnerForeignPK, filter.OwnerUUID)
	if err != nil {
		return nil, err
	}

	ids := []MultiPassUUID{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		mp := raw.(*Multipass)
		ids = append(ids, mp.UUID)
	}
	return ids, nil
}

func (r *MultipassRepository) SetExtension(ext *Extension) error {
	if ext.OwnerType != ServiceAccountType {
		return fmt.Errorf("multipass extension is suppoted only for serviceaacounts, got type %q", ext.OwnerType)
	}
	obj, err := r.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[ObjectOrigin]*Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return r.save(obj)
}

func (r *MultipassRepository) UnsetExtension(origin ObjectOrigin, uuid MultiPassUUID) error {
	obj, err := r.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return r.save(obj)
}
