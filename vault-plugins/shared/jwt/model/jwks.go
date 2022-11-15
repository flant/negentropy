package model

import (
	"fmt"
	"time"

	hcmemdb "github.com/hashicorp/go-memdb"
	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	JWKSType = "jwks" // also, memdb schema name
	PK       = "id"
)

type JSONWebKey struct {
	jose.JSONWebKey
	GenerateTime time.Time `json:"generate_time"`
	StartTime    time.Time `json:"start_time"`
	EndLifeTime  time.Time `json:"end_life_time"`
}

type JSONWebKeySet struct {
	Keys []*JSONWebKey `json:"keys"`
}

type JWKS struct {
	ID     string
	KeySet *JSONWebKeySet
}

func (p *JWKS) ObjType() string {
	return JWKSType
}

func (p *JWKS) ObjId() string {
	return p.ID
}

func JWKSTables() map[string]*hcmemdb.TableSchema {
	return map[string]*hcmemdb.TableSchema{
		JWKSType: {
			Name: JWKSType,
			Indexes: map[string]*hcmemdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &hcmemdb.StringFieldIndex{
						Field: "ID",
					},
				},
			},
		},
	}
}

type JWKSRepo struct {
	db          *io.MemoryStoreTxn
	publisherId string
}

func NewJWKSRepo(db *io.MemoryStoreTxn, publisherId string) *JWKSRepo {
	return &JWKSRepo{
		db:          db,
		publisherId: publisherId,
	}
}

func (r *JWKSRepo) DeleteOwn() error {
	err := r.db.Delete(JWKSType, &JWKS{
		ID: r.publisherId,
	})
	if err != nil {
		return fmt.Errorf("DeleteOwn:  r.publisherId=%s, err=%w", r.publisherId, err)
	}
	return nil
}

func (r *JWKSRepo) UpdateOwn(keySet *JSONWebKeySet) error {
	jwks, err := r.GetOwn()
	if err != nil {
		return err
	}

	if jwks == nil {
		jwks = &JWKS{
			ID: r.publisherId,
		}
	}

	jwks.KeySet = keySet

	return r.db.Insert(JWKSType, jwks)
}

func (r *JWKSRepo) GetOwn() (*JWKS, error) {
	jwksRaw, err := r.db.First(JWKSType, PK, r.publisherId)
	if err != nil {
		return nil, err
	}

	if jwksRaw == nil {
		return nil, nil
	}

	jwks, ok := jwksRaw.(*JWKS)
	if !ok {
		return nil, fmt.Errorf("cannot cast to JWKS")
	}

	return jwks, nil
}

func (r *JWKSRepo) GetSet() ([]jose.JSONWebKey, error) {
	keys := make([]jose.JSONWebKey, 0)
	err := r.Iter(func(j *JWKS) (bool, error) {
		for _, k := range j.KeySet.Keys {
			keys = append(keys, k.JSONWebKey)
		}

		return true, nil
	})

	return keys, err
}

func (r *JWKSRepo) Iter(action func(*JWKS) (bool, error)) error {
	iter, err := r.db.Get(JWKSType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*JWKS)
		next, err := action(t)
		if err != nil {
			return err
		}

		if !next {
			return nil
		}
	}

	return nil
}
