package backend

import (
	"context"
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
)

type uuidGenerator struct{}

func (g *uuidGenerator) GenerateID() string {
	return genUUID()
}

func genUUID() string {
	id, err := uuid.GenerateUUID()
	if err != nil {
		id = genUUID()
	}
	return id
}

// Schema contains specific entity type data
type Schema interface {
	Type() string
	Fields() map[string]*framework.FieldSchema
	Validate(*framework.FieldData) error
	GenerateID() string

	SyncTopics() []Topic
	ParseEntry(*logical.StorageEntry) (Data, error)
	ParseData(data *framework.FieldData) (Data, error)
}

var ErrNotFound = fmt.Errorf("not found")

type Repository struct {
	schema Schema
}

func (r *Repository) Put(ctx context.Context, storage logical.Storage, key string, data Data) error {
	buf, err := jsonutil.EncodeJSON(data)
	if err != nil {
		return errwrap.Wrapf("json encoding failed: {{err}}", err)
	}

	entry := &logical.StorageEntry{
		Key:   key,
		Value: buf,
	}

	err = storage.Put(ctx, entry)
	return err
}

func (r *Repository) Get(ctx context.Context, storage logical.Storage, key string) (Data, error) {
	entry, err := storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, ErrNotFound
	}

	data, err := r.schema.ParseEntry(entry)
	if err != nil {
		return nil, err
	}

	return data, nil
}
