package model

import "github.com/flant/negentropy/vault-plugins/shared/memdb"

const FeatureFlagType = "feature_flag" // also, memdb schema name

type FeatureFlag struct {
	memdb.ArchivableImpl

	Name FeatureFlagName `json:"name"` // PK
}

type TenantFeatureFlag struct {
	FeatureFlag `json:",inline"`

	EnabledForNewProjects bool `json:"enabled_for_new"`
}

func (f *FeatureFlag) ObjType() string {
	return FeatureFlagType
}

func (f *FeatureFlag) ObjId() string {
	return f.Name
}
