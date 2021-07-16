package model

import (
	"github.com/hashicorp/go-memdb"
)

const (
	ProjectForeignPK = "project_uuid"
)

func ProjectSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ProjectType: {
				Name: ProjectType,
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

//go:generate go run gen_repository.go -type Project -parentType Tenant
type Project struct {
	UUID       ProjectUUID `json:"uuid"` // PK
	TenantUUID TenantUUID  `json:"tenant_uuid"`
	Version    string      `json:"resource_version"`
	Identifier string      `json:"identifier"`

	FeatureFlags []FeatureFlag `json:"feature_flags"`
}
