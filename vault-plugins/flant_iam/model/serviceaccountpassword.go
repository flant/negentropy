package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	ServiceAccountPasswordType = "service_account_password" // also, memdb schema name
)

func ServiceAccountPasswordSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ServiceAccountPasswordType: {
				Name: ServiceAccountPasswordType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
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

type ServiceAccountPassword struct {
	UUID       string `json:"uuid"` // PK
	TenantUUID string `json:"tenant_uuid"`
	OwnerUUID  string `json:"owner_uuid"`

	Description string `json:"description"`

	ValidTill int64    `json:"valid_till"`
	CIDRs     []string `json:"allowed_cidrs"`
	Roles     []string `json:"allowed_roles" `

	Secret string `json:"secret,omitempty" sensitive:""`
}

func (t *ServiceAccountPassword) ObjType() string {
	return ServiceAccountPasswordType
}

func (t *ServiceAccountPassword) ObjId() string {
	return t.UUID
}

func (t *ServiceAccountPassword) Marshal(includeSensitive bool) ([]byte, error) {
	obj := t
	if !includeSensitive {
		t := OmitSensitive(*t).(ServiceAccountPassword)
		obj = &t
	}
	return jsonutil.EncodeJSON(obj)
}

func (t *ServiceAccountPassword) Unmarshal(data []byte) error {
	return jsonutil.DecodeJSON(data, t)
}

/*
Параметры:
	- uuid – идентификатор пароля
	- description – комментарий о том, где это используется и зачем (чтобы потом можно было вспомнить).
	- allowed_roles, allowed_cidrs – аналогично multipass’ам.
	- password_ttl – аналогично multipass_ttl (именно multipass_ttl, а не multipass_jwt_ttl).
Особенности
	- При создании нового пароля метод возвращает, в качестве результата, сгенерированный пароль в открытом виде (пароль длинный и страшный)
	- При создании на основании переданного password_ttl создается параметр valid_till с конкретным временем окончания пароля.
	- Пару password_uuid + password_secret можно использовать для логина в плагине flant_iam_auth.

*/
type ServiceAccountPasswordRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewServiceAccountPasswordRepository(tx *io.MemoryStoreTxn) *ServiceAccountPasswordRepository {
	return &ServiceAccountPasswordRepository{db: tx}
}

func (r *ServiceAccountPasswordRepository) validate(p *ServiceAccountPassword) error {
	tenantRepo := NewTenantRepository(r.db)
	_, err := tenantRepo.GetByID(p.TenantUUID)
	if err != nil {
		return err
	}

	repo := NewServiceAccountRepository(r.db)
	owner, err := repo.GetByID(p.OwnerUUID)
	if err != nil {
		return err
	}
	if owner.TenantUUID != p.TenantUUID {
		return ErrNotFound
	}

	return nil
}

func (r *ServiceAccountPasswordRepository) save(p *ServiceAccountPassword) error {
	return r.db.Insert(ServiceAccountPasswordType, p)
}

func (r *ServiceAccountPasswordRepository) delete(filter *ServiceAccountPassword) error {
	return r.db.Delete(ServiceAccountPasswordType, filter)
}

func (r *ServiceAccountPasswordRepository) Create(p *ServiceAccountPassword) error {
	err := r.validate(p)
	if err != nil {
		return err
	}
	return r.save(p)
}

func (r *ServiceAccountPasswordRepository) Delete(filter *ServiceAccountPassword) error {
	err := r.validate(filter)
	if err != nil {
		return err
	}
	return r.delete(filter)
}

func (r *ServiceAccountPasswordRepository) Get(filter *ServiceAccountPassword) (*ServiceAccountPassword, error) {
	err := r.validate(filter)
	if err != nil {
		return nil, err
	}
	return r.GetByID(filter.UUID)
}

func (r *ServiceAccountPasswordRepository) GetByID(id string) (*ServiceAccountPassword, error) {
	raw, err := r.db.First(ServiceAccountPasswordType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	password := raw.(*ServiceAccountPassword)
	return password, nil
}

func (r *ServiceAccountPasswordRepository) List(filter *ServiceAccountPassword) ([]string, error) {
	err := r.validate(filter)
	if err != nil {
		return nil, err
	}

	iter, err := r.db.Get(ServiceAccountPasswordType, OwnerForeignPK, filter.OwnerUUID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		p := raw.(*ServiceAccountPassword)
		ids = append(ids, p.UUID)
	}
	return ids, nil
}
