package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func ContactSchema() map[string]*hcmemdb.TableSchema {
	return map[string]*hcmemdb.TableSchema{
		model.ContactType: {
			Name: model.ContactType,
			Indexes: map[string]*hcmemdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &hcmemdb.UUIDFieldIndex{
						Field: "UUID",
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
				"identifier": {
					Name: "identifier",
					Indexer: &hcmemdb.StringFieldIndex{
						Field: "Identifier",
					},
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
	_, err := r.GetByID(contact.UUID)
	if err != nil {
		return err
	}
	return r.save(contact)
}

func (r *ContactRepository) Delete(id model.ContactUUID, archiveMark memdb.ArchiveMark) error {
	contact, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if contact.IsDeleted() {
		return consts.ErrIsArchived
	}
	contact.Timestamp = archiveMark.Timestamp
	contact.Hash = archiveMark.Hash
	return r.Update(contact)
}

func (r *ContactRepository) List(clientUUID model.ClientUUID, showArchived bool) ([]*model.Contact, error) {
	iter, err := r.db.Get(model.ContactType, ClientForeignPK, clientUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.Contact{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Contact)
		if showArchived || !obj.Archived() {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *ContactRepository) ListIDs(clientID model.ClientUUID, showArchived bool) ([]model.ContactUUID, error) {
	objs, err := r.List(clientID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.ContactUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ContactRepository) Iter(action func(*model.Contact) (bool, error)) error {
	iter, err := r.db.Get(model.ContactType, PK)
	if err != nil {
		return err
	}

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
	if !contact.Archived() {
		return nil, consts.ErrIsNotArchived
	}
	contact.Timestamp = 0
	contact.Hash = 0
	err = r.Update(contact)
	if err != nil {
		return nil, err
	}
	return contact, nil
}
