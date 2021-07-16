package model

import (
	"github.com/hashicorp/go-memdb"
)

func RoleBindingApprovalSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			RoleBindingApprovalType: {
				Name: RoleBindingApprovalType,
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
					RoleBindingForeignPK: {
						Name: RoleBindingForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "RoleBindingUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}

//go:generate go run gen_repository.go -type RoleBindingApproval -parentType RoleBinding
type RoleBindingApproval struct {
	UUID            RoleBindingApprovalUUID `json:"uuid"` // PK
	TenantUUID      TenantUUID              `json:"tenant_uuid"`
	RoleBindingUUID RoleBindingUUID         `json:"role_binding_uuid"`
	Version         string                  `json:"resource_version"`

	Users           []UserUUID           `json:"user_uuids"`
	Groups          []GroupUUID          `json:"group_uuids"`
	ServiceAccounts []ServiceAccountUUID `json:"service_account_uuids"`

	RequiredVotes int  `json:"required_votes"`
	RequireMFA    bool `json:"require_mfa"`

	RequireUniqueApprover bool `json:"require_unique_approver"`
}

func (r *RoleBindingApprovalRepository) UpdateOrCreate(ra *RoleBindingApproval) error {
	stored, err := r.GetByID(ra.UUID)
	if err != nil {
		if err == ErrNotFound {
			ra.Version = NewResourceVersion()
			return r.save(ra)
		}
		return err
	}

	// Validate
	err = r.validate(stored, ra)
	if err != nil {
		return err
	}
	ra.Version = NewResourceVersion()

	// Update
	return r.save(ra)
}

func (r *RoleBindingApprovalRepository) validate(stored, newRa *RoleBindingApproval) error {
	if stored.TenantUUID != newRa.TenantUUID {
		return ErrNotFound
	}
	if stored.RoleBindingUUID != newRa.RoleBindingUUID {
		return ErrNotFound
	}
	if stored.Version != newRa.Version {
		return ErrBadVersion
	}

	return nil
}
