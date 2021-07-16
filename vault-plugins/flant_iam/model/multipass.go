package model

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
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

type MultipassOwnerType string

const (
	MultipassOwnerServiceAccount MultipassOwnerType = "service_account"
	MultipassOwnerUser           MultipassOwnerType = "user"
)

//go:generate go run gen_repository.go -type Multipass -parentType Owner
type Multipass struct {
	UUID       MultipassUUID      `json:"uuid"` // PK
	TenantUUID TenantUUID         `json:"tenant_uuid"`
	OwnerUUID  OwnerUUID          `json:"owner_uuid"`
	OwnerType  MultipassOwnerType `json:"owner_type"`

	Description string        `json:"description"`
	TTL         time.Duration `json:"ttl"`
	MaxTTL      time.Duration `json:"max_ttl"`
	CIDRs       []string      `json:"allowed_cidrs"`
	Roles       []RoleName    `json:"allowed_roles" `

	ValidTill int64  `json:"valid_till"`
	Salt      string `json:"salt,omitempty" sensitive:""`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"-"`
}

const MultipassType = "multipass" // also, memdb schema name

func (u *Multipass) ObjType() string {
	return MultipassType
}

func (u *Multipass) ObjId() string {
	return u.UUID
}

type MultipassRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewMultipassRepository(tx *io.MemoryStoreTxn) *MultipassRepository {
	return &MultipassRepository{db: tx}
}

func (r *MultipassRepository) save(multipass *Multipass) error {
	return r.db.Insert(MultipassType, multipass)
}

func (r *MultipassRepository) Create(multipass *Multipass) error {
	return r.save(multipass)
}

func (r *MultipassRepository) GetRawByID(id MultipassUUID) (interface{}, error) {
	raw, err := r.db.First(MultipassType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *MultipassRepository) GetByID(id MultipassUUID) (*Multipass, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*Multipass), err
}

func (r *MultipassRepository) Update(multipass *Multipass) error {
	_, err := r.GetByID(multipass.UUID)
	if err != nil {
		return err
	}
	return r.save(multipass)
}

func (r *MultipassRepository) Delete(id MultipassUUID) error {
	multipass, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(MultipassType, multipass)
}

func (r *MultipassRepository) List(ownerUUID OwnerUUID) ([]*Multipass, error) {
	iter, err := r.db.Get(MultipassType, OwnerForeignPK, ownerUUID)
	if err != nil {
		return nil, err
	}

	list := []*Multipass{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Multipass)
		list = append(list, obj)
	}
	return list, nil
}

func (r *MultipassRepository) ListIDs(ownerID OwnerUUID) ([]MultipassUUID, error) {
	objs, err := r.List(ownerID)
	if err != nil {
		return nil, err
	}
	ids := make([]MultipassUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *MultipassRepository) Iter(action func(*Multipass) (bool, error)) error {
	iter, err := r.db.Get(MultipassType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Multipass)
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

func (r *MultipassRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	multipass := &Multipass{}
	err := json.Unmarshal(data, multipass)
	if err != nil {
		return err
	}

	return r.save(multipass)
}
