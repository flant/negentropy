package model

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-memdb"
	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	JWKSType = "jwks" // also, memdb schema name
	idKey = "id"
)

type JSONWebKey struct {
	jose.JSONWebKey
	GenerateTime time.Time `json:"generate_time"`
	EndLifeTime  time.Time `json:"end_life_time"`
}

type JSONWebKeySet struct {
	Keys []*JSONWebKey `json:"keys"`
}

type JWKS struct {
	ID   string
	KeySet *JSONWebKeySet
}

func (p *JWKS) ObjType() string {
	return JWKSType
}

func (p *JWKS) ObjId() string {
	return p.ID
}

func JWKSSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			JWKSType: {
				Name: JWKSType,
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


type JWKSRepo struct {
	db        *io.MemoryStoreTxn
	tableName string
	publisherId string
}

func NewJWKSRepo(db *io.MemoryStoreTxn, publisherId string) *JWKSRepo {
	return &JWKSRepo{
		db:        db,
		tableName: JWKSType,
		publisherId: publisherId,
	}
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

	return r.db.Insert(r.tableName, jwks)
}

func (r *JWKSRepo) GetOwn() (*JWKS, error){
	jwksRaw, err := r.db.First(r.tableName, idKey, r.publisherId)
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
	iter, err := r.db.Get(r.tableName, idKey)
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