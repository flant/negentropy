package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const (
	RoleBindingInServicePackIndex = "rb_in_service_pack_index"
)

func ServicePackSchema() *memdb.DBSchema {
	pkIndexer := &hcmemdb.CompoundIndex{
		Indexes: []hcmemdb.Indexer{
			&hcmemdb.UUIDFieldIndex{Field: "ProjectUUID"},
			&hcmemdb.StringFieldIndex{Field: "Name", Lowercase: true},
		},
	}

	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.ServicePackType: {
				Name: model.ServicePackType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:    PK,
						Unique:  true,
						Indexer: pkIndexer,
					},
					RoleBindingInServicePackIndex: {
						Name:         RoleBindingInServicePackIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "Rolebindings",
						},
					},
					ProjectForeignPK: {
						Name: ProjectForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "ProjectUUID",
							Lowercase: true,
						},
					},
					repo.IdentitySharingForeignPK: {
						Name:         repo.IdentitySharingForeignPK,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "IdentitySharings",
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.ServicePackType: {
				{OriginalDataTypeFieldName: "ProjectUUID", RelatedDataType: iam_model.ProjectType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "Rolebindings", RelatedDataType: iam_model.RoleBindingType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "IdentitySharings", RelatedDataType: iam_model.IdentitySharingType, RelatedDataTypeFieldIndexName: PK},
			},
		},
	}
}

type ServicePackRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewServicePackRepository(tx *io.MemoryStoreTxn) *ServicePackRepository {
	return &ServicePackRepository{db: tx}
}

func (r *ServicePackRepository) save(servicePack *model.ServicePack) error {
	return r.db.Insert(model.ServicePackType, servicePack)
}

func (r *ServicePackRepository) Create(servicePack *model.ServicePack) error {
	return r.save(servicePack)
}

func (r *ServicePackRepository) getRawByID(projectUUID iam_model.ProjectUUID,
	servicePackName model.ServicePackName) (interface{}, error) {
	raw, err := r.db.First(model.ServicePackType, PK, projectUUID, servicePackName)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *ServicePackRepository) GetByID(projectUUID iam_model.ProjectUUID,
	servicePackName model.ServicePackName) (*model.ServicePack, error) {
	raw, err := r.getRawByID(projectUUID, servicePackName)
	if raw == nil {
		return nil, err
	}
	g := raw.(*model.ServicePack)
	return g, err
}

func (r *ServicePackRepository) Update(servicePack *model.ServicePack) error {
	_, err := r.GetByID(servicePack.ProjectUUID, servicePack.Name)
	if err != nil {
		return err
	}
	return r.save(servicePack)
}

func (r *ServicePackRepository) Delete(projectUUID iam_model.ProjectUUID,
	servicePackName model.ServicePackName, archiveMark memdb.ArchiveMark) error {
	servicePack, err := r.GetByID(projectUUID, servicePackName)
	if err != nil {
		return err
	}
	if servicePack.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(model.ServicePackType, servicePack, archiveMark)
}

func (r *ServicePackRepository) List(projectUUID iam_model.ProjectUUID, showArchived bool) ([]*model.ServicePack, error) {
	iter, err := r.db.Get(model.ServicePackType, ProjectForeignPK, projectUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.ServicePack{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.ServicePack)
		if showArchived || obj.NotArchived() {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *ServicePackRepository) Sync(objID string, data []byte) error {
	group := &model.ServicePack{}
	err := json.Unmarshal(data, group)
	if err != nil {
		return err
	}

	return r.save(group)
}

func (r *ServicePackRepository) ListForIdentitySharing(identitySharingUUID iam_model.IdentitySharingUUID,
	showArchived bool) ([]*model.ServicePack, error) {
	iter, err := r.db.Get(model.ServicePackType, repo.IdentitySharingForeignPK, identitySharingUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.ServicePack{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.ServicePack)
		if showArchived || obj.NotArchived() {
			list = append(list, obj)
		}
	}
	return list, nil
}
