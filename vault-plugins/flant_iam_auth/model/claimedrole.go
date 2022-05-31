package model

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

type RoleClaim struct {
	Role        model.RoleName         `json:"role"`
	Claim       map[string]interface{} `json:"claim,omitempty"`
	TenantUUID  model.TenantUUID       `json:"tenant_uuid,omitempty"`
	ProjectUUID model.ProjectUUID      `json:"project_uuid,omitempty"`
}
