package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
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

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

const RoleBindingApprovalType = "role_binding_approval" // also, memdb schema name

func (r *RoleBindingApproval) isDeleted() bool {
	return r.ArchivingTimestamp != 0
}

func (r *RoleBindingApproval) ObjType() string {
	return RoleBindingApprovalType
}

func (r *RoleBindingApproval) ObjId() string {
	return r.UUID
}

type RoleBindingApprovalRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleBindingApprovalRepository(tx *io.MemoryStoreTxn) *RoleBindingApprovalRepository {
	return &RoleBindingApprovalRepository{db: tx}
}

func (r *RoleBindingApprovalRepository) save(appr *RoleBindingApproval) error {
	return r.db.Insert(RoleBindingApprovalType, appr)
}

func (r *RoleBindingApprovalRepository) Create(appr *RoleBindingApproval) error {
	return r.save(appr)
}

func (r *RoleBindingApprovalRepository) GetRawByID(id RoleBindingApprovalUUID) (interface{}, error) {
	raw, err := r.db.First(RoleBindingApprovalType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *RoleBindingApprovalRepository) GetByID(id RoleBindingApprovalUUID) (*RoleBindingApproval, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*RoleBindingApproval), err
}

func (r *RoleBindingApprovalRepository) Update(appr *RoleBindingApproval) error {
	_, err := r.GetByID(appr.UUID)
	if err != nil {
		return err
	}
	return r.save(appr)
}

func (r *RoleBindingApprovalRepository) Delete(id RoleBindingApprovalUUID,
	archivingTimestamp UnixTime, archivingHash int64) error {
	appr, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if appr.isDeleted() {
		return ErrIsArchived
	}
	appr.ArchivingTimestamp = archivingTimestamp
	appr.ArchivingHash = archivingHash
	return r.Update(appr)
}

func (r *RoleBindingApprovalRepository) List(rbUUID RoleBindingUUID,
	showArchived bool) ([]*RoleBindingApproval, error) {
	iter, err := r.db.Get(RoleBindingApprovalType, RoleBindingForeignPK, rbUUID)
	if err != nil {
		return nil, err
	}

	list := []*RoleBindingApproval{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*RoleBindingApproval)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *RoleBindingApprovalRepository) ListIDs(rbID RoleBindingUUID,
	showArchived bool) ([]RoleBindingApprovalUUID, error) {
	objs, err := r.List(rbID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]RoleBindingApprovalUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *RoleBindingApprovalRepository) Iter(action func(*RoleBindingApproval) (bool, error)) error {
	iter, err := r.db.Get(RoleBindingApprovalType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*RoleBindingApproval)
		next, err := action(obj)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
}

func (r *RoleBindingApprovalRepository) Sync(_ string, data []byte) error {
	appr := &RoleBindingApproval{}
	err := json.Unmarshal(data, appr)
	if err != nil {
		return err
	}

	return r.save(appr)
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
