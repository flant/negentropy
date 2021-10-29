package repo

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func ClientSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			model.ClientType: {
				Name: model.ClientType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					"identifier": {
						Name:   "identifier",
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field:     "Identifier",
							Lowercase: true,
						},
					},
					"version": {
						Name: "version",
						Indexer: &memdb.StringFieldIndex{
							Field: "Version",
						},
					},
				},
			},
		},
	}
}

type ClientRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewClientRepository(tx *io.MemoryStoreTxn) *ClientRepository {
	return &ClientRepository{db: tx}
}

func (r *ClientRepository) save(client *model.Client) error {
	return r.db.Insert(model.ClientType, client)
}

func (r *ClientRepository) Create(client *model.Client) error {
	return r.save(client)
}

func (r *ClientRepository) GetRawByID(id model.ClientUUID) (interface{}, error) {
	raw, err := r.db.First(model.ClientType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *ClientRepository) GetByID(id model.ClientUUID) (*model.Client, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Client), err
}

func (r *ClientRepository) Update(client *model.Client) error {
	_, err := r.GetByID(client.UUID)
	if err != nil {
		return err
	}
	return r.save(client)
}

func (r *ClientRepository) Delete(id model.ClientUUID, archivingTimestamp model.UnixTime, archivingHash int64) error {
	client, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if client.IsDeleted() {
		return model.ErrIsArchived
	}
	client.ArchivingTimestamp = archivingTimestamp
	client.ArchivingHash = archivingHash
	return r.Update(client)
}

func (r *ClientRepository) List(showArchived bool) ([]*model.Client, error) {
	iter, err := r.db.Get(model.ClientType, PK)
	if err != nil {
		return nil, err
	}

	list := []*model.Client{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Client)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *ClientRepository) ListIDs(showArchived bool) ([]model.ClientUUID, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.ClientUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ClientRepository) Iter(action func(*model.Client) (bool, error)) error {
	iter, err := r.db.Get(model.ClientType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Client)
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

func (r *ClientRepository) Sync(_ string, data []byte) error {
	client := &model.Client{}
	err := json.Unmarshal(data, client)
	if err != nil {
		return err
	}

	return r.save(client)
}

func (r *ClientRepository) Restore(id model.ClientUUID) (*model.Client, error) {
	client, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if client.ArchivingTimestamp == 0 {
		return nil, model.ErrIsNotArchived
	}
	client.ArchivingTimestamp = 0
	client.ArchivingHash = 0
	err = r.Update(client)
	if err != nil {
		return nil, err
	}
	return client, nil
}
