package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

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
	UUID       UserUUID   `json:"uuid"` // PK
	TenantUUID TenantUUID `json:"tenant_uuid"`
	Version    string     `json:"resource_version"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"-"`

	Identifier     string `json:"identifier"`
	FullIdentifier string `json:"full_identifier"` // calculated <identifier>@<tenant_identifier>

	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`

	Email            string   `json:"email"`
	AdditionalEmails []string `json:"additional_emails"`

	MobilePhone      string   `json:"mobile_phone"`
	AdditionalPhones []string `json:"additional_phones"`
}

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

func (r *UserRepository) Delete(id UserUUID) error {
	user, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(UserType, user)
}

func (r *UserRepository) List(tenantID TenantUUID) ([]*User, error) {
	iter, err := r.db.Get(UserType, TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	list := []*User{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*User)
		list = append(list, u)
	}
	return list, nil
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

func (r *UserRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	user := &User{}
	err := json.Unmarshal(data, user)
	if err != nil {
		return err
	}

	return r.save(user)
}
