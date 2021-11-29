package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func ContactSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.ContactType: {
				Name: model.ContactType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UserUUID",
						},
					},
					ClientForeignPK: {
						Name: ClientForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					"version": {
						Name: "version",
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Version",
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.ContactType: {
				{
					OriginalDataTypeFieldName:     "UserUUID",
					RelatedDataType:               iam_model.UserType,
					RelatedDataTypeFieldIndexName: PK,
				},
				{
					OriginalDataTypeFieldName:     "TenantUUID",
					RelatedDataType:               iam_model.TenantType,
					RelatedDataTypeFieldIndexName: PK,
				},
			},
		},
		CascadeDeletes: map[string][]memdb.Relation{
			model.ContactType: {
				{
					OriginalDataTypeFieldName:     "UserUUID",
					RelatedDataType:               iam_model.UserType,
					RelatedDataTypeFieldIndexName: PK,
				},
			},
		},
	}
}

type ContactRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics

}

func NewContactRepository(tx *io.MemoryStoreTxn) *ContactRepository {
	return &ContactRepository{db: tx}
}

func (r *ContactRepository) save(contact *model.Contact) error {
	return r.db.Insert(model.ContactType, contact)
}

func (r *ContactRepository) Create(contact *model.Contact) error {
	return r.save(contact)
}

func (r *ContactRepository) GetRawByID(id model.ContactUUID) (interface{}, error) {
	raw, err := r.db.First(model.ContactType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *ContactRepository) GetByID(id model.ContactUUID) (*model.Contact, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Contact), err
}

func (r *ContactRepository) Update(contact *model.Contact) error {
	stored, err := r.GetByID(contact.UserUUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	return r.save(contact)
}

func (r *ContactRepository) Delete(id model.ContactUUID, archiveMark memdb.ArchiveMark) error {
	contact, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if contact.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(model.ContactType, contact, archiveMark)
}

func (r *ContactRepository) List(clientUUID model.ClientUUID, showArchived bool) ([]*model.Contact, error) {
	iter, err := r.db.Get(model.ContactType, ClientForeignPK, clientUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.Contact{}
	err = r.Iter(iter, func(contact *model.Contact) (bool, error) {
		if showArchived || contact.NotArchived() {
			list = append(list, contact)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *ContactRepository) ListIDs(clientUUID model.ClientUUID, showArchived bool) ([]model.ContactUUID, error) {
	iter, err := r.db.Get(model.ContactType, ClientForeignPK, clientUUID)
	if err != nil {
		return nil, err
	}
	ids := []model.ContactUUID{}
	err = r.Iter(iter, func(contact *model.Contact) (bool, error) {
		if showArchived || contact.NotArchived() {
			ids = append(ids, contact.UserUUID)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *ContactRepository) Iter(iter hcmemdb.ResultIterator, action func(*model.Contact) (bool, error)) error {
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Contact)
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

func (r *ContactRepository) Sync(_ string, data []byte) error {
	contact := &model.Contact{}
	err := json.Unmarshal(data, contact)
	if err != nil {
		return err
	}

	return r.save(contact)
}

func (r *ContactRepository) Restore(id model.ContactUUID) (*model.Contact, error) {
	contact, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if contact.NotArchived() {
		return nil, err
	}
	err = r.db.CascadeRestore(model.ContactType, contact)
	if err != nil {
		return nil, err
	}
	return contact, nil
}
