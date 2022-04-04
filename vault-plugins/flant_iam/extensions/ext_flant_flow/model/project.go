package model

import (
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

// Project is stored at memdb as a regular iam.Project with extension
type Project struct {
	memdb.ArchiveMark

	UUID       ProjectUUID `json:"uuid"` // PK
	TenantUUID ClientUUID  `json:"tenant_uuid"`
	Version    string      `json:"resource_version"`
	Identifier string      `json:"identifier"`

	FeatureFlags []iam_model.FeatureFlagName `json:"feature_flags"`

	Origin consts.ObjectOrigin `json:"origin,omitempty"`

	// Extensions map[consts.ObjectOrigin]*Extension `json:"extensions"`
	ServicePacks map[ServicePackName]ServicePackCFG `json:"service_packs"`
}
