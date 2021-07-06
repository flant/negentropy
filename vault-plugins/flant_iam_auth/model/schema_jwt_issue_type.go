package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	JWTIssueTypeType = "jwt_type" // also, memdb schema name
)

func JWTIssueTypeSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			JWTIssueTypeType: {
				Name: JWTIssueTypeType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					ByName: {
						Name: ByName,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
				},
			},
		},
	}
}

type JWTIssueType struct {
	UUID       string `json:"uuid"` // ID
	Name       string `json:"name"`

	TTL time.Duration `json:"ttl"`
	OptionsSchema string `json:"options_schema"`
	// TODO rego policy
}

func (p *JWTIssueType) ObjType() string {
	return JWTIssueTypeType
}

func (p *JWTIssueType) ObjId() string {
	return p.UUID
}

func (p *JWTIssueType) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(p)
}

func (p *JWTIssueType) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, p)
	return err
}
