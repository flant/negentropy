package model

type FeatureFlag struct {
	Name FeatureFlagName `json:"name"` // PK

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

type TenantFeatureFlag struct {
	FeatureFlag `json:",inline"`

	EnabledForNewProjects bool `json:"enabled_for_new"`
}

const FeatureFlagType = "feature_flag" // also, memdb schema name

func (f *FeatureFlag) IsDeleted() bool {
	return f.ArchivingTimestamp != 0
}

func (f *FeatureFlag) ObjType() string {
	return FeatureFlagType
}

func (f *FeatureFlag) ObjId() string {
	return f.Name
}
