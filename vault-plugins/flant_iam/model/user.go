package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func UserSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			UserType: {
				Name: UserType,
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
					"version": {
						Name: "version",
						Indexer: &memdb.StringFieldIndex{
							Field: "Version",
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &memdb.StringFieldIndex{
							Field: "Identifier",
						},
					},
				},
			},
		},
	}
}

type User struct {
	UUID       UserUUID   `json:"uuid"` // PK
	TenantUUID TenantUUID `json:"tenant_uuid"`
	Version    string     `json:"resource_version"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"extensions"`

	Identifier     string `json:"identifier"`
	FullIdentifier string `json:"full_identifier"` // calculated <identifier>@<tenant_identifier>

	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`

	Email            string   `json:"email"`
	AdditionalEmails []string `json:"additional_emails"`

	MobilePhone      string   `json:"mobile_phone"`
	AdditionalPhones []string `json:"additional_phones"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

const UserType = "user" // also, memdb schema name

func (u *User) ObjType() string {
	return UserType
}

func (u *User) ObjId() string {
	return u.UUID
}

type UserRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewUserRepository(tx *io.MemoryStoreTxn) *UserRepository {
	return &UserRepository{db: tx}
}

func (r *UserRepository) save(user *User) error {
	return r.db.Insert(UserType, user)
}

func (r *UserRepository) Create(user *User) error {
	return r.save(user)
}

func (r *UserRepository) GetRawByID(id UserUUID) (interface{}, error) {
	raw, err := r.db.First(UserType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *UserRepository) GetByID(id UserUUID) (*User, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*User), err
}

func (r *UserRepository) Update(user *User) error {
	_, err := r.GetByID(user.UUID)
	if err != nil {
		return err
	}
	return r.save(user)
}

func (r *UserRepository) Delete(id UserUUID, archivingTimestamp UnixTime, archivingHash int64) error {
	user, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if user.ArchivingTimestamp != 0 {
		return ErrIsArchived
	}
	user.ArchivingTimestamp = archivingTimestamp
	user.ArchivingHash = archivingHash
	return r.Update(user)
}

func (r *UserRepository) List(tenantUUID TenantUUID, showArchived bool) ([]*User, error) {
	iter, err := r.db.Get(UserType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*User{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*User)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *UserRepository) ListIDs(tenantID TenantUUID, showArchived bool) ([]UserUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]UserUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *UserRepository) Iter(action func(*User) (bool, error)) error {
	iter, err := r.db.Get(UserType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*User)
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

func (r *UserRepository) Sync(_ string, data []byte) error {
	user := &User{}
	err := json.Unmarshal(data, user)
	if err != nil {
		return err
	}

	return r.save(user)
}

func (r *UserRepository) Restore(id UserUUID) (*User, error) {
	user, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user.ArchivingTimestamp == 0 {
		return nil, ErrIsNotArchived
	}
	user.ArchivingTimestamp = 0
	user.ArchivingHash = 0
	err = r.Update(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
