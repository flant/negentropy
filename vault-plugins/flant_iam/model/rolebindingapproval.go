package model

import (
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	RoleBindingApprovalType = "role_binding_approval"
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

func (t *RoleBindingApproval) ObjType() string {
	return RoleBindingApprovalType
}

func (t *RoleBindingApproval) ObjId() string {
	return t.UUID
}

type RoleBindingApprovalRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewRoleBindingApprovalRepository(tx *io.MemoryStoreTxn) *RoleBindingApprovalRepository {
	return &RoleBindingApprovalRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *RoleBindingApprovalRepository) save(ra *RoleBindingApproval) error {
	return r.db.Insert(RoleBindingApprovalType, ra)
}

func (r *RoleBindingApprovalRepository) delete(id RoleBindingApprovalUUID) error {
	ra, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(RoleBindingApprovalType, ra)
}

func (r *RoleBindingApprovalRepository) GetByID(id RoleBindingApprovalUUID) (*RoleBindingApproval, error) {
	raw, err := r.db.First(RoleBindingApprovalType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	ra := raw.(*RoleBindingApproval)
	return ra, nil
}

func (r *RoleBindingApprovalRepository) Create(ra *RoleBindingApproval) error {
	ra.Version = NewResourceVersion()

	// Update
	return r.save(ra)
}

func (r *RoleBindingApprovalRepository) Update(ra *RoleBindingApproval) error {
	stored, err := r.GetByID(ra.UUID)
	if err != nil {
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

func (r *RoleBindingApprovalRepository) Delete(id RoleBindingApprovalUUID) error {
	return r.delete(id)
}

func (r *RoleBindingApprovalRepository) Iter(action func(approval *RoleBindingApproval) (bool, error)) error {
	iter, err := r.db.Get(RoleBindingApprovalType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*RoleBindingApproval)
		next, err := action(t)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
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
