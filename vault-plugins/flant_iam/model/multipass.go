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

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

const MultipassType = "multipass" // also, memdb schema name

func (m *Multipass) IsDeleted() bool {
	return m.ArchivingTimestamp != 0
}

func (m *Multipass) ObjType() string {
	return MultipassType
}

func (m *Multipass) ObjId() string {
	return m.UUID
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

func (r *MultipassRepository) Delete(id MultipassUUID, archivingTimestamp UnixTime, archivingHash int64) error {
	multipass, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if multipass.IsDeleted() {
		return ErrIsArchived
	}
	multipass.ArchivingTimestamp = archivingTimestamp
	multipass.ArchivingHash = archivingHash
	return r.Update(multipass)
}

func (r *MultipassRepository) List(ownerUUID OwnerUUID, showArchived bool) ([]*Multipass, error) {
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
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *MultipassRepository) ListIDs(ownerID OwnerUUID, showArchived bool) ([]MultipassUUID, error) {
	objs, err := r.List(ownerID, showArchived)
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

func (r *MultipassRepository) Sync(_ string, data []byte) error {
	multipass := &Multipass{}
	err := json.Unmarshal(data, multipass)
	if err != nil {
		return err
	}

	return r.save(multipass)
}
