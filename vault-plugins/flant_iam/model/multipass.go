package model

import (
	"net/http"
	"time"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	MultipassType  = "token" // also, memdb schema name
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
							Field: "OwnerUUID",
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
	UUID        string             `json:"uuid"` // PK
	TenantUUID  string             `json:"tenant_uuid"`
	OwnerUUID   string             `json:"owner_uuid"`
	OwnerType   MultipassOwnerType `json:"owner_type"`
	Description string             `json:"description"`
	TTL         time.Duration      `json:"ttl"`
	MaxTTL      time.Duration      `json:"max_ttl"`
	ValidTill   int64              `json:"valid_till"`
	CIDRs       []string           `json:"allowed_cidrs"`
	Roles       []string           `json:"allowed_roles" `
	Salt        string             `json:"salt" sensitive:""`
}

func (t *Multipass) ObjType() string {
	return MultipassType
}

func (t *Multipass) ObjId() string {
	return t.UUID
}

func (t *Multipass) Marshal(includeSensitive bool) ([]byte, error) {
	obj := t
	if !includeSensitive {
		t := OmitSensitive(*t).(Multipass)
		obj = &t
	}
	return jsonutil.EncodeJSON(obj)
}

func (t *Multipass) Unmarshal(data []byte) error {
	return jsonutil.DecodeJSON(data, t)
}


func (b *userBackend) handleMultipassCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		var (
			ttl       = time.Duration(data.Get("ttl").(int)) * time.Second
			maxTTL    = time.Duration(data.Get("max_ttl").(int)) * time.Second
			validTill = time.Now().Add(ttl).Unix()
		)

		multipass := &Multipass{
			UUID:        uuid.New(),
			TenantUUID:  data.Get("tenant_uuid").(string),
			OwnerUUID:   data.Get("owner_uuid").(string),
			OwnerType:   MultipassOwnerUser,
			Description: data.Get("description").(string),
			TTL:         ttl,
			MaxTTL:      maxTTL,
			ValidTill:   validTill,
			CIDRs:       data.Get("allowed_cidrs").([]string),
			Roles:       data.Get("allowed_roles").([]string),
		}

		tx := b.storage.Txn(true)
		repo := NewMultipassRepository(tx)

		err := repo.Create(multipass)
		if err == ErrNotFound {
			return responseNotFound(req, "something in the path")
		}
		if err != nil {
			return nil, err
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, multipass, http.StatusCreated)
	}
}

func (b *userBackend) handleMultipassDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		filter := &Multipass{
			UUID:       data.Get("uuid").(string),
			TenantUUID: data.Get("tenant_uuid").(string),
			OwnerUUID:  data.Get("owner_uuid").(string),
			OwnerType:  MultipassOwnerUser,
		}

		tx := b.storage.Txn(true)
		repo := NewMultipassRepository(tx)

		err := repo.Delete(filter)
		if err != ErrNotFound {
			return responseNotFound(req, "something in the path")
		}
		if err != nil {
			return nil, err
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func (b *userBackend) handleMultipassRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		filter := &Multipass{
			UUID:       data.Get("uuid").(string),
			TenantUUID: data.Get("tenant_uuid").(string),
			OwnerUUID:  data.Get("owner_uuid").(string),
			OwnerType:  MultipassOwnerUser,
		}

		tx := b.storage.Txn(false)
		repo := NewMultipassRepository(tx)

		mp, err := repo.Get(filter)
		if err != ErrNotFound {
			return responseNotFound(req, "something in the path")
		}
		if err != nil {
			return nil, err
		}

		return responseWithData(mp)
	}
}

func (b *userBackend) handleMultipassList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		filter := &Multipass{
			TenantUUID: data.Get("tenant_uuid").(string),
			OwnerUUID:  data.Get("owner_uuid").(string),
			OwnerType:  MultipassOwnerUser,
		}

		tx := b.storage.Txn(false)
		repo := NewMultipassRepository(tx)

		ids, err := repo.List(filter)
		if err != ErrNotFound {
			return responseNotFound(req, "something in the path")
		}
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuids": ids,
			},
		}

		return resp, nil
	}
}

type MultipassRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewMultipassRepository(tx *io.MemoryStoreTxn) *MultipassRepository {
	return &MultipassRepository{db: tx}
}

func (r *MultipassRepository) validate(multipass *Multipass) error {
	tenantRepo := NewTenantRepository(r.db)
	_, err := tenantRepo.GetByID(multipass.TenantUUID)
	if err != nil {
		return err
	}

	if multipass.OwnerType == MultipassOwnerUser {
		repo := NewUserRepository(r.db)
		owner, err := repo.GetByID(multipass.OwnerUUID)
		if err != nil {
			return err
		}
		if owner.TenantUUID != multipass.TenantUUID {
			return ErrNotFound
		}
	}

	if multipass.OwnerType == MultipassOwnerServiceAccount {
		repo := NewServiceAccountRepository(r.db)
		owner, err := repo.GetByID(multipass.OwnerUUID)
		if err != nil {
			return err
		}
		if owner.TenantUUID != multipass.TenantUUID {
			return ErrNotFound
		}
	}

	return nil
}

func (r *MultipassRepository) Create(mp *Multipass) error {
	err := r.validate(mp)
	if err != nil {
		return err
	}

	return r.db.Insert(MultipassType, mp)
}

func (r *MultipassRepository) Delete(filter *Multipass) error {
	err := r.validate(filter)
	if err != nil {
		return err
	}

	return r.db.Delete(MultipassType, filter)
}

func (r *MultipassRepository) Get(filter *Multipass) (*Multipass, error) {
	err := r.validate(filter)
	if err != nil {
		return nil, err
	}

	return r.GetByID(filter.UUID)
}

func (r *MultipassRepository) GetByID(id string) (*Multipass, error) {
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

func (r *MultipassRepository) List(filter *Multipass) ([]string, error) {
	err := r.validate(filter)
	if err != nil {
		return nil, err
	}

	iter, err := r.db.Get(MultipassType, OwnerForeignPK, filter.OwnerUUID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
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
