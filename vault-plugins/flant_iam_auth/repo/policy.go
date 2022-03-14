package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const (
	RoleInPolicyIndex = "role_in_policy"
)

func PolicySchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.PolicyType: {
				Name: model.PolicyType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Name",
						},
					},
					RoleInPolicyIndex: {
						Name:         RoleInPolicyIndex,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "Roles",
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.PolicyType: {
				{OriginalDataTypeFieldName: "Roles", RelatedDataType: iam.RoleType, RelatedDataTypeFieldIndexName: ID},
			},
		},
	}
}

type PolicyRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewPolicyRepository(tx *io.MemoryStoreTxn) *PolicyRepository {
	return &PolicyRepository{db: tx}
}

func (r *PolicyRepository) save(policy *model.Policy) error {
	return r.db.Insert(model.PolicyType, policy)
}

func (r *PolicyRepository) Create(policy *model.Policy) error {
	return r.save(policy)
}

func (r *PolicyRepository) GetRawByID(name model.PolicyName) (interface{}, error) {
	raw, err := r.db.First(model.PolicyType, ID, name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *PolicyRepository) GetByID(name model.PolicyName) (*model.Policy, error) {
	raw, err := r.GetRawByID(name)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Policy), err
}

func (r *PolicyRepository) Update(policy *model.Policy) error {
	_, err := r.GetByID(policy.Name)
	if err != nil {
		return err
	}
	return r.save(policy)
}

func (r *PolicyRepository) Delete(name model.PolicyName, archiveMark memdb.ArchiveMark) error {
	policy, err := r.GetByID(name)
	if err != nil {
		return err
	}
	if policy.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(model.PolicyType, policy, archiveMark)
}

func (r *PolicyRepository) List(showArchived bool) ([]*model.Policy, error) {
	iter, err := r.db.Get(model.PolicyType, ID)
	if err != nil {
		return nil, err
	}

	list := []*model.Policy{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Policy)
		if showArchived || obj.NotArchived() {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *PolicyRepository) ListIDs(showArchived bool) ([]model.PolicyName, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.PolicyName, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *PolicyRepository) Iter(action func(*model.Policy) (bool, error)) error {
	iter, err := r.db.Get(model.PolicyType, ID)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Policy)
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

func (r *PolicyRepository) Sync(_ string, data []byte) error {
	policy := &model.Policy{}
	err := json.Unmarshal(data, policy)
	if err != nil {
		return err
	}

	return r.save(policy)
}

func (r *PolicyRepository) Restore(name model.PolicyName) (*model.Policy, error) {
	policy, err := r.GetByID(name)
	if err != nil {
		return nil, err
	}
	if policy.NotArchived() {
		return nil, consts.ErrIsNotArchived
	}
	err = r.db.Restore(model.PolicyType, policy)
	if err != nil {
		return nil, err
	}
	return policy, nil
}
