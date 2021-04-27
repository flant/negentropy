package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	ProjectType = "project" // also, memdb schema name

)

func ProjectSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ProjectType: {
				Name: ProjectType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name:   TenantForeignPK,
						Unique: true,
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

type Project struct {
	UUID       string `json:"uuid"` // ID
	TenantUUID string `json:"tenant_uuid"`
	Version    string `json:"resource_version"`
	Identifier string `json:"identifier"`
}

func (p *Project) ObjType() string {
	return ProjectType
}

func (p *Project) ObjId() string {
	return p.UUID
}

func (p *Project) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(p)
}

func (p *Project) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, p)
	return err
}
