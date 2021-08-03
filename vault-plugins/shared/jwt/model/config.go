package model

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	JWTConfigType  = "jwt_config"  // also, memdb schema name
	configStoreKey = JWTConfigType // also, memdb schema name
)

type Config struct {
	Issuer         string        `json:"issuer" structs:"issuer" mapstructure:"issuer"`
	OwnAudience    string        `json:"multipass_audience" structs:"multipass_audience" mapstructure:"multipass_audience"`
	RotationPeriod time.Duration `json:"rotation_period" structs:"rotation_period" mapstructure:"rotation_period"`

	PreliminaryAnnouncePeriod time.Duration `json:"preliminary_announce_period" structs:"preliminary_announce_period" mapstructure:"preliminary_announce_period"`
}

// we need store one entry in table
// for guarantee id we do wrapper, that id set always internal in repo
// and user does not change it if not use memdb directly :-)
type configPairTableEntity struct {
	ID     string  `json:"id"`
	Config *Config `json:"config"`
}

func (p *configPairTableEntity) ObjType() string {
	return JWTConfigType
}

func (p *configPairTableEntity) ObjId() string {
	return p.ID
}

type ConfigRepo struct {
	db *io.MemoryStoreTxn
	// using mem store for announce new public key/pair
	storeKey string
}

func ConfigSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			JWTConfigType: {
				Name: JWTConfigType,
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

func NewConfigRepo(db *io.MemoryStoreTxn) *ConfigRepo {
	return &ConfigRepo{
		db:       db,
		storeKey: configStoreKey,
	}
}

func (s *ConfigRepo) Put(config *Config) error {
	wrap := &configPairTableEntity{
		ID:     s.storeKey,
		Config: config,
	}

	return s.db.Insert(JWTConfigType, wrap)
}

func (s *ConfigRepo) Get() (*Config, error) {
	entryRaw, err := s.db.First(JWTConfigType, idKey, s.storeKey)
	if err != nil {
		return nil, err
	}

	if entryRaw == nil {
		def := DefaultConfig()
		err := s.Put(def)
		if err != nil {
			return nil, err
		}

		return def, nil
	}

	entry := entryRaw.(*configPairTableEntity)

	return entry.Config, nil
}

func HandleRestoreConfig(db *memdb.Txn, o interface{}) error {
	entry, ok := o.(*configPairTableEntity)
	if !ok {
		return fmt.Errorf("does not restore jwt config. cannot cast")
	}

	if entry.ID != configStoreKey {
		return fmt.Errorf("does not restore jwt config. incorrect id %s. need %s", entry.ID, configStoreKey)
	}

	return db.Insert(JWTConfigType, entry)
}

func DefaultConfig() *Config {
	return &Config{
		Issuer:                    "https://auth.negentropy.flant.com/",
		OwnAudience:               "",
		PreliminaryAnnouncePeriod: 24 * time.Hour,
		RotationPeriod:            336 * time.Hour,
	}
}
