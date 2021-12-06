package model

import "github.com/flant/negentropy/vault-plugins/shared/memdb"

const RoleBindingApprovalType = "role_binding_approval" // also, memdb schema name

type RoleBindingApproval struct {
	memdb.ArchiveMark

	UUID            RoleBindingApprovalUUID `json:"uuid"` // PK
	TenantUUID      TenantUUID              `json:"tenant_uuid"`
	RoleBindingUUID RoleBindingUUID         `json:"role_binding_uuid"`
	Version         string                  `json:"resource_version"`

	Users           []UserUUID           `json:"user_uuids"`
	Groups          []GroupUUID          `json:"group_uuids"`
	ServiceAccounts []ServiceAccountUUID `json:"service_account_uuids"`
	Approvers       []MemberNotation     `json:"approvers"`

	RequiredVotes int  `json:"required_votes"`
	RequireMFA    bool `json:"require_mfa"`

	RequireUniqueApprover bool `json:"require_unique_approver"`
}

func (r *RoleBindingApproval) ObjType() string {
	return RoleBindingApprovalType
}

func (r *RoleBindingApproval) ObjId() string {
	return r.UUID
}

// FixApprovers remove from members invalid links, if some removed, returns true
func (r *RoleBindingApproval) FixApprovers() bool {
	return FixMembers(&r.Approvers, r.Users, r.Groups, r.ServiceAccounts)
}
