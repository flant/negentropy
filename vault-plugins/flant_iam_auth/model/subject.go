package model

import iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"

// Subject is a representation of iam.ServiceAccount or iam.User
type Subject struct {
	// user or service_account
	Type string `json:"type"`
	// UUID of iam.ServiceAccount or iam.User
	UUID string `json:"uuid"`
	//  tenant_uuid of subject
	TenantUUID iam.TenantUUID `json:"tenant_uuid"`
}
