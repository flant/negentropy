package model

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
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
	UUID       ServiceAccountPasswordUUID `json:"uuid"` // PK
	TenantUUID TenantUUID                 `json:"tenant_uuid"`
	OwnerUUID  OwnerUUID                  `json:"owner_uuid"`

	Description string `json:"description"`

	CIDRs []string   `json:"allowed_cidrs"`
	Roles []RoleName `json:"allowed_roles" `

	TTL       time.Duration `json:"ttl"`
	ValidTill int64         `json:"valid_till"` // calculates from TTL on creation

	Secret string `json:"secret,omitempty" sensitive:""` // generates on creation
}

const ServiceAccountPasswordType = "service_account_password" // also, memdb schema name

func (u *ServiceAccountPassword) ObjType() string {
	return ServiceAccountPasswordType
}

func (u *ServiceAccountPassword) ObjId() string {
	return u.UUID
}

type ServiceAccountPasswordRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewServiceAccountPasswordRepository(tx *io.MemoryStoreTxn) *ServiceAccountPasswordRepository {
	return &ServiceAccountPasswordRepository{db: tx}
}

func (r *ServiceAccountPasswordRepository) save(sap *ServiceAccountPassword) error {
	return r.db.Insert(ServiceAccountPasswordType, sap)
}

func (r *ServiceAccountPasswordRepository) Create(sap *ServiceAccountPassword) error {
	return r.save(sap)
}

func (r *ServiceAccountPasswordRepository) GetRawByID(id ServiceAccountPasswordUUID) (interface{}, error) {
	raw, err := r.db.First(ServiceAccountPasswordType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *ServiceAccountPasswordRepository) GetByID(id ServiceAccountPasswordUUID) (*ServiceAccountPassword, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*ServiceAccountPassword), err
}

func (r *ServiceAccountPasswordRepository) Update(sap *ServiceAccountPassword) error {
	_, err := r.GetByID(sap.UUID)
	if err != nil {
		return err
	}
	return r.save(sap)
}

func (r *ServiceAccountPasswordRepository) Delete(id ServiceAccountPasswordUUID) error {
	sap, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(ServiceAccountPasswordType, sap)
}

func (r *ServiceAccountPasswordRepository) List(ownerUUID OwnerUUID) ([]*ServiceAccountPassword, error) {
	iter, err := r.db.Get(ServiceAccountPasswordType, OwnerForeignPK, ownerUUID)
	if err != nil {
		return nil, err
	}

	list := []*ServiceAccountPassword{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*ServiceAccountPassword)
		list = append(list, obj)
	}
	return list, nil
}

func (r *ServiceAccountPasswordRepository) ListIDs(ownerID OwnerUUID) ([]ServiceAccountPasswordUUID, error) {
	objs, err := r.List(ownerID)
	if err != nil {
		return nil, err
	}
	ids := make([]ServiceAccountPasswordUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ServiceAccountPasswordRepository) Iter(action func(*ServiceAccountPassword) (bool, error)) error {
	iter, err := r.db.Get(ServiceAccountPasswordType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*ServiceAccountPassword)
		next, err := action(obj)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
}

func (r *ServiceAccountPasswordRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	sap := &ServiceAccountPassword{}
	err := json.Unmarshal(data, sap)
	if err != nil {
		return err
	}

	return r.save(sap)
}
