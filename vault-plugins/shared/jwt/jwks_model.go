package jwt

import (
	"crypto/ed25519"

	"github.com/hashicorp/go-memdb"
)

const (
	JWKSType = "jwks" // also, memdb schema name
)

func JWKSSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			JWKSType: {
				Name: JWKSType,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
				},
			},
		},
	}
}

type JWKS struct {
	UUID   string `json:"uuid"`
	Pubkey *ed25519.PublicKey
}

func (p *JWKS) ObjType() string {
	return JWKSType
}

func (p *JWKS) ObjId() string {
	return p.UUID
}
