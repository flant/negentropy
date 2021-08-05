package model

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	JWTStateType     = "jwt_state" // also, memdb schema name
	jwtStateStoreKey = JWTStateType
)

type KeyPair struct {
	PrivateKeys *JSONWebKeySet `json:"private_keys"`
	PublicKeys  *JSONWebKeySet `json:"public_keys"`
}

// we need store one entry in table
// for guarantee id we do wrapper, that id set always internal in repo
// and user does not change it if not use memdb directly :-)
type state struct {
	ID               string    `json:"id"`
	Pair             *KeyPair  `json:"key_pair"`
	Enabled          bool      `json:"enabled"`
	LastRotationTime time.Time `json:"last_rotation_time"`
}

func (p *state) ObjType() string {
	return JWTStateType
}

func (p *state) ObjId() string {
	return p.ID
}

type StateRepo struct {
	db *io.MemoryStoreTxn
	// using mem store for announce new public key/pair
	storeKey string
}

func StateSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			JWTStateType: {
				Name: JWTStateType,
				Indexes: map[string]*memdb.IndexSchema{
					idKey: {
						Name:   idKey,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "ID",
						},
					},
				},
			},
		},
	}
}

func NewStateRepo(db *io.MemoryStoreTxn) *StateRepo {
	return &StateRepo{
		db:       db,
		storeKey: jwtStateStoreKey,
	}
}

func (s *StateRepo) SetKeyPair(pair *KeyPair) error {
	st, err := s.get()
	if err != nil {
		return err
	}

	st.Pair = pair

	return s.put(st)
}

func (s *StateRepo) GetKeyPair() (*KeyPair, error) {
	st, err := s.get()
	if err != nil {
		return nil, err
	}

	return st.Pair, nil
}

func (s *StateRepo) SetEnabled(f bool) error {
	st, err := s.get()
	if err != nil {
		return err
	}

	st.Enabled = f

	return s.put(st)
}

func (s *StateRepo) IsEnabled() (bool, error) {
	st, err := s.get()
	if err != nil {
		return false, err
	}

	return st.Enabled, nil
}

func (s *StateRepo) GetLastRotationTime() (time.Time, error) {
	st, err := s.get()
	if err != nil {
		return time.Time{}, err
	}

	return st.LastRotationTime, nil
}

func (s *StateRepo) SetLastRotationTime(t time.Time) error {
	st, err := s.get()
	if err != nil {
		return err
	}

	st.LastRotationTime = t

	return s.put(st)
}

func HandleRestoreState(db *memdb.Txn, o interface{}) error {
	entry, ok := o.(*state)
	if !ok {
		return fmt.Errorf("does not restore jwt keypair. cannot cast")
	}

	if entry.ID != jwtStateStoreKey {
		return fmt.Errorf("does not restore jwt keypair. incorrect id %s. need %s", entry.ID, jwtStateStoreKey)
	}

	return db.Insert(JWTStateType, entry)
}

func (s *StateRepo) put(st *state) error {
	st.ID = s.storeKey
	return s.db.Insert(JWTStateType, st)
}

func (s *StateRepo) get() (*state, error) {
	entryRaw, err := s.db.First(JWTStateType, idKey, s.storeKey)
	if err != nil {
		return nil, err
	}

	if entryRaw == nil {
		return &state{ID: s.storeKey}, nil
	}

	return entryRaw.(*state), nil
}