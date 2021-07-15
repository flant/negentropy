package model

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/go-memdb"

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

type MultipassOwnerType string

const (
	MultipassOwnerServiceAccount MultipassOwnerType = "service_account"
	MultipassOwnerUser           MultipassOwnerType = "user"
)

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

func (t *Multipass) ObjType() string {
	return MultipassType
}

func (t *Multipass) ObjId() string {
	return t.UUID
}

type MultipassRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewMultipassRepository(tx *io.MemoryStoreTxn) *MultipassRepository {
	return &MultipassRepository{db: tx}
}

func (r *MultipassRepository) save(mp *Multipass) error {
	return r.db.Insert(MultipassType, mp)
}

func (r *MultipassRepository) Delete(id string) error {
	mp, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(MultipassType, mp)
}

func (r *MultipassRepository) Create(mp *Multipass) error {
	return r.save(mp)
}

func (r *MultipassRepository) Update(mp *Multipass) error {
	_, err := r.GetByID(mp.UUID)
	if err != nil {
		return err
	}
	return r.save(mp)
}

func (r *MultipassRepository) GetByID(id MultipassUUID) (*Multipass, error) {
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

func (r *MultipassRepository) List(oid OwnerUUID) ([]*Multipass, error) {
	iter, err := r.db.Get(MultipassType, OwnerForeignPK, oid)
	if err != nil {
		return nil, err
	}

	list := []*Multipass{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		mp := raw.(*Multipass)
		list = append(list, mp)
	}
	return list, nil
}

func (r *MultipassRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	mp := &Multipass{}
	err := json.Unmarshal(data, mp)
	if err != nil {
		return err
	}

	return r.save(mp)
}
