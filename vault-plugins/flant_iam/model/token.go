package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	TokenType      = "token" // also, memdb schema name
	OwnerForeignPK = "owner_uuid"
)

func TokenSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			TokenType: {
				Name: TokenType,
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

type TokenOwner string

const (
	TokenOwnerServiceAccount TokenOwner = "service_account"
	TokenOwnerUser           TokenOwner = "user"
)

type Token struct {
	UUID        string        `json:"uuid"` // PK
	TenantUUID  string        `json:"tenant_uuid"`
	OwnerUUID   string        `json:"owner_uuid"`
	OwnerType   TokenOwner    `json:"owner_type"`
	Description string        `json:"description"`
	TTL         time.Duration `json:"ttl"`
	MaxTTL      time.Duration `json:"max_ttl"`
	ValidTill   int64         `json:"valid_till"`
	CIDRs       []string      `json:"allowed_cidrs"`
	Roles       []string      `json:"allowed_roles" `
	Salt        string        `json:"salt" sensitive:""`
}

func (t *Token) ObjType() string {
	return TokenType
}

func (t *Token) ObjId() string {
	return t.UUID
}

func (t *Token) Marshal(includeSensitive bool) ([]byte, error) {
	obj := t
	if !includeSensitive {
		t := OmitSensitive(*t).(Token)
		obj = &t
	}
	return jsonutil.EncodeJSON(obj)
}

func (t *Token) Unmarshal(data []byte) error {
	return jsonutil.DecodeJSON(data, t)
}
