package repo

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func RoleBindingApprovalSchema() map[string]*memdb.TableSchema {
	return map[string]*memdb.TableSchema{
		model.RoleBindingApprovalType: {
			Name: model.RoleBindingApprovalType,
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
	}
}

type RoleBindingApprovalRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleBindingApprovalRepository(tx *io.MemoryStoreTxn) *RoleBindingApprovalRepository {
	return &RoleBindingApprovalRepository{db: tx}
}

func (r *RoleBindingApprovalRepository) save(appr *model.RoleBindingApproval) error {
	return r.db.Insert(model.RoleBindingApprovalType, appr)
}

func (r *RoleBindingApprovalRepository) Create(appr *model.RoleBindingApproval) error {
	return r.save(appr)
}

func (r *RoleBindingApprovalRepository) GetRawByID(id model.RoleBindingApprovalUUID) (interface{}, error) {
	raw, err := r.db.First(model.RoleBindingApprovalType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *RoleBindingApprovalRepository) GetByID(id model.RoleBindingApprovalUUID) (*model.RoleBindingApproval, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.RoleBindingApproval), err
}

func (r *RoleBindingApprovalRepository) Update(appr *model.RoleBindingApproval) error {
	_, err := r.GetByID(appr.UUID)
	if err != nil {
		return err
	}
	return r.save(appr)
}

func (r *RoleBindingApprovalRepository) Delete(id model.RoleBindingApprovalUUID,
	archivingTimestamp model.UnixTime, archivingHash int64) error {
	appr, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if appr.IsDeleted() {
		return model.ErrIsArchived
	}
	appr.ArchivingTimestamp = archivingTimestamp
	appr.ArchivingHash = archivingHash
	return r.Update(appr)
}

func (r *RoleBindingApprovalRepository) List(rbUUID model.RoleBindingUUID,
	showArchived bool) ([]*model.RoleBindingApproval, error) {
	iter, err := r.db.Get(model.RoleBindingApprovalType, RoleBindingForeignPK, rbUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.RoleBindingApproval{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.RoleBindingApproval)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *RoleBindingApprovalRepository) ListIDs(rbID model.RoleBindingUUID,
	showArchived bool) ([]model.RoleBindingApprovalUUID, error) {
	objs, err := r.List(rbID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.RoleBindingApprovalUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *RoleBindingApprovalRepository) Iter(action func(*model.RoleBindingApproval) (bool, error)) error {
	iter, err := r.db.Get(model.RoleBindingApprovalType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.RoleBindingApproval)
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
	appr := &model.RoleBindingApproval{}
	err := json.Unmarshal(data, appr)
	if err != nil {
		return err
	}

	return r.save(appr)
}

func (r *RoleBindingApprovalRepository) UpdateOrCreate(ra *model.RoleBindingApproval) error {
	stored, err := r.GetByID(ra.UUID)
	if err != nil {
		if err == model.ErrNotFound {
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

func (r *RoleBindingApprovalRepository) validate(stored, newRa *model.RoleBindingApproval) error {
	if stored.TenantUUID != newRa.TenantUUID {
		return model.ErrNotFound
	}
	if stored.RoleBindingUUID != newRa.RoleBindingUUID {
		return model.ErrNotFound
	}
	if stored.Version != newRa.Version {
		return model.ErrBadVersion
	}

	return nil
}
