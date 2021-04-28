package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	FeatureFlagType = "feature_flag" // also, memdb schema name
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

type FeatureFlag struct {
	Name string `json:"name"` // PK
}

func (t *FeatureFlag) ObjType() string {
	return FeatureFlagType
}

func (t *FeatureFlag) ObjId() string {
	return t.Name
}

func (t *FeatureFlag) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(t)
}

func (t *FeatureFlag) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, t)
	return err
}
