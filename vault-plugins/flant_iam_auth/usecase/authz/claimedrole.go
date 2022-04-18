package authz

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

type RoleClaim struct {
	Role        model.RoleName         `json:"role"`
	Claim       map[string]interface{} `json:"claim"`
	TenantUUID  model.TenantUUID       `json:"tenant_uuid"`
	ProjectUUID model.ProjectUUID      `json:"project_uuid"`
}
