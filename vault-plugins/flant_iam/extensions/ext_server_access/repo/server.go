package repo

import (
	hcmemdb "github.com/hashicorp/go-memdb"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func ServerSchema() *memdb.DBSchema {
	var serverIdentifierMultiIndexer []hcmemdb.Indexer

	tenantUUIDIndex := &hcmemdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, tenantUUIDIndex)

	projectUUIDIndex := &hcmemdb.StringFieldIndex{
		Field:     "ProjectUUID",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, projectUUIDIndex)

	serverIdentifierIndex := &hcmemdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, serverIdentifierIndex)

	var tenantProjectMultiIndexer []hcmemdb.Indexer
	tenantProjectMultiIndexer = append(tenantProjectMultiIndexer, tenantUUIDIndex)
	tenantProjectMultiIndexer = append(tenantProjectMultiIndexer, projectUUIDIndex)

	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			ext_model.ServerType: {
				Name: ext_model.ServerType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					iam_repo.PK: {
						Name:   iam_repo.PK,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					iam_repo.TenantForeignPK: {
						Name: iam_repo.TenantForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					iam_repo.ProjectForeignPK: {
						Name: iam_repo.ProjectForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "ProjectUUID",
							Lowercase: true,
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &hcmemdb.CompoundIndex{
							Indexes: serverIdentifierMultiIndexer,
						},
						Unique: true,
					},
					"tenant_project": {
						Name: "tenant_project",
						Indexer: &hcmemdb.CompoundIndex{
							Indexes: tenantProjectMultiIndexer,
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			ext_model.ServerType: {
				{OriginalDataTypeFieldName: "TenantUUID", RelatedDataType: iam_model.TenantType, RelatedDataTypeFieldIndexName: iam_repo.PK},
				{OriginalDataTypeFieldName: "ProjectUUID", RelatedDataType: iam_model.ProjectType, RelatedDataTypeFieldIndexName: iam_repo.PK},
				// {OriginalDataTypeFieldName: "MultipassUUID", RelatedDataType: iam_model.MultipassType, RelatedDataTypeFieldIndexName: iam_repo.PK}, may have not multipass
			},
		},
		CascadeDeletes: map[string][]memdb.Relation{
			iam_model.TenantType:  {{OriginalDataTypeFieldName: "UUID", RelatedDataType: ext_model.ServerType, RelatedDataTypeFieldIndexName: iam_repo.TenantForeignPK}},
			iam_model.ProjectType: {{OriginalDataTypeFieldName: "UUID", RelatedDataType: ext_model.ServerType, RelatedDataTypeFieldIndexName: iam_repo.ProjectForeignPK}},
		},
	}
}

type ServerRepository struct {
	db *io.MemoryStoreTxn
}

func NewServerRepository(tx *io.MemoryStoreTxn) *ServerRepository {
	return &ServerRepository{
		db: tx,
	}
}

func (r *ServerRepository) Create(server *ext_model.Server) error {
	return r.db.Insert(ext_model.ServerType, server)
}

func (r *ServerRepository) GetByUUID(id string) (*ext_model.Server, error) {
	raw, err := r.db.First(ext_model.ServerType, iam_repo.PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}

	server := raw.(*ext_model.Server)
	return server, nil
}

func (r *ServerRepository) GetByID(tenant_uuid, project_uuid, id string) (*ext_model.Server, error) {
	raw, err := r.db.First(ext_model.ServerType, "identifier", tenant_uuid, project_uuid, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}

	server := raw.(*ext_model.Server)
	return server, nil
}

func (r *ServerRepository) Update(server *ext_model.Server) error {
	_, err := r.GetByUUID(server.UUID)
	if err != nil {
		return err
	}
	return r.db.Insert(ext_model.ServerType, server)
}

func (r *ServerRepository) Delete(uuid string, archiveMark memdb.ArchiveMark) error {
	server, err := r.GetByUUID(uuid)
	if err != nil {
		return err
	}
	if server.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(ext_model.ServerType, server, archiveMark)
}

func (r *ServerRepository) List(tenantID, projectID string, showArchived bool) ([]*ext_model.Server, error) {
	var (
		iter hcmemdb.ResultIterator
		err  error
	)

	switch {
	case tenantID != "" && projectID != "":
		iter, err = r.db.Get(ext_model.ServerType, "tenant_project", tenantID, projectID)

	case tenantID != "":
		iter, err = r.db.Get(ext_model.ServerType, iam_repo.TenantForeignPK, tenantID)

	case projectID != "":
		iter, err = r.db.Get(ext_model.ServerType, iam_repo.ProjectForeignPK, projectID)

	default:
		iter, err = r.db.Get(ext_model.ServerType, iam_repo.PK)
	}
	if err != nil {
		return nil, err
	}

	ids := make([]*ext_model.Server, 0)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*ext_model.Server)
		if showArchived || u.NotArchived() {
			ids = append(ids, u)
		}
	}
	return ids, nil
}
