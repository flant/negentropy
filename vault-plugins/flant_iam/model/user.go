package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	UserType = "user" // also, memdb schema name

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
	UUID           string     `json:"uuid"` // PK
	TenantUUID     string     `json:"tenant_uuid"`
	Version        string     `json:"resource_version"`
	Identifier     string     `json:"identifier"`
	FullIdentifier string     `json:"full_identifier"` // calculated <identifier>@<tenant_identifier>
	Email          string     `json:"email"`
	Extension      *Extension `json:"extension"`
}

func (u *User) ObjType() string {
	return UserType
}

func (u *User) ObjId() string {
	return u.UUID
}

func (u *User) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(u)
}

func (u *User) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, u)
	return err
}

type UserRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewUserRepository(tx *io.MemoryStoreTxn) *UserRepository {
	return &UserRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *UserRepository) Create(user *User) error {
	tenant, err := r.tenantRepo.GetById(user.TenantUUID)
	if err != nil {
		return err
	}

	user.Version = NewResourceVersion()
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier

	err = r.db.Insert(UserType, user)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) GetById(id string) (*User, error) {
	raw, err := r.db.First(UserType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	user := raw.(*User)
	return user, nil
}

func (r *UserRepository) Update(user *User) error {
	stored, err := r.GetById(user.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != user.TenantUUID {
		return ErrNotFound
	}
	if stored.Version != user.Version {
		return ErrVersionMismatch
	}
	user.Version = NewResourceVersion()

	// Update

	tenant, err := r.tenantRepo.GetById(user.TenantUUID)
	if err != nil {
		return err
	}
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier

	err = r.db.Insert(UserType, user)
	if err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) Delete(id string) error {
	user, err := r.GetById(id)
	if err != nil {
		return err
	}

	return r.db.Delete(UserType, user)
}

func (r *UserRepository) List(tenantID string) ([]string, error) {
	iter, err := r.db.Get(UserType, TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*User)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}

func (r *UserRepository) DeleteByTenant(tenantUUID string) error {
	_, err := r.db.DeleteAll(UserType, TenantForeignPK, tenantUUID)
	return err
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
		t := raw.(*User)
		next, err := action(t)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
}
