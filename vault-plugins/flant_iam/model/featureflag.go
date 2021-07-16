package model

import (
	"github.com/hashicorp/go-memdb"
)

func FeatureFlagSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			FeatureFlagType: {
				Name: FeatureFlagType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
				},
			},
		},
	}
}

//go:generate go run gen_repository.go -type FeatureFlag -IDsuffix Name
type FeatureFlag struct {
	Name FeatureFlagName `json:"name"` // PK
}

type TenantFeatureFlag struct {
	FeatureFlag `json:",inline"`

	EnabledForNewProjects bool `json:"enabled_for_new"`
}
