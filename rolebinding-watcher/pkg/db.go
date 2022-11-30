package pkg

import (
	hcmemdb "github.com/hashicorp/go-memdb"

	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const userIndex = "user_index"
const roleIndex = "role_Index"

func UserEffectiveRolesSchema() *memdb.DBSchema {
	userIndexer := &hcmemdb.StringFieldIndex{
		Field:     "UserUUID",
		Lowercase: true,
	}
	roleNameIndexer := &hcmemdb.StringFieldIndex{
		Field:     "RoleName",
		Lowercase: true,
	}

	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			UserEffectiveRolesType: {
				Name: UserEffectiveRolesType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					iam_repo.PK: {
						Name:    iam_repo.PK,
						Unique:  true,
						Indexer: &hcmemdb.CompoundIndex{Indexes: []hcmemdb.Indexer{userIndexer, roleNameIndexer}},
					},
					userIndex: {
						Name:    userIndex,
						Indexer: userIndexer,
					},
					roleIndex: {
						Name:    roleIndex,
						Indexer: roleNameIndexer,
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			UserEffectiveRolesType: {
				{OriginalDataTypeFieldName: "UserUUID", RelatedDataType: model.UserType, RelatedDataTypeFieldIndexName: iam_repo.PK},
				{OriginalDataTypeFieldName: "RoleName", RelatedDataType: model.RoleType, RelatedDataTypeFieldIndexName: iam_repo.PK},
			},
		},
		CascadeDeletes: map[string][]memdb.Relation{},
	}
}

type UserEffectiveRolesRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewUserEffectiveRolesRepository(txn *io.MemoryStoreTxn) *UserEffectiveRolesRepository {
	return &UserEffectiveRolesRepository{db: txn}
}

func (r *UserEffectiveRolesRepository) GetByKey(key UserEffectiveRolesKey) (*UserEffectiveRoles, error) {
	raw, err := r.db.First(UserEffectiveRolesType, iam_repo.PK, key.UserUUID, key.RoleName)
	if raw == nil {
		return nil, err
	}
	result := raw.(*UserEffectiveRoles)
	return result, err
}

func (r *UserEffectiveRolesRepository) Save(userEffectiveRoles *UserEffectiveRoles) error {
	return r.db.Insert(UserEffectiveRolesType, userEffectiveRoles)
}

func (r *UserEffectiveRolesRepository) Delete(userEffectiveRoles *UserEffectiveRoles) error {
	return r.db.Delete(UserEffectiveRolesType, userEffectiveRoles)
}
