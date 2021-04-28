package jwtauth

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
)

type PrefixStorage struct {
	prefix        string
	parentStorage logical.Storage
}

type PrefixStorageRequestFactory func(request *logical.Request) *PrefixStorage

func (s *PrefixStorage) List(ctx context.Context, _ string) ([]string, error) {
	return s.parentStorage.List(ctx, s.prefix)
}

func (s *PrefixStorage) AllKeys(ctx context.Context) ([]string, error) {
	return s.parentStorage.List(ctx, s.prefix)
}

func (s *PrefixStorage) Get(ctx context.Context, name string) (*logical.StorageEntry, error) {
	return s.parentStorage.Get(ctx, s.fullPath(name))
}

func (s *PrefixStorage) Put(ctx context.Context, entry *logical.StorageEntry) error {
	return s.parentStorage.Put(ctx, entry)
}

func (s *PrefixStorage) PutEntry(ctx context.Context, name string, v interface{}) error {
	entry, err := logical.StorageEntryJSON(s.fullPath(name), v)
	if err != nil {
		return err
	}

	return s.Put(ctx, entry)
}

func (s *PrefixStorage) Delete(ctx context.Context, name string) error {
	return s.parentStorage.Delete(ctx, s.fullPath(name))
}

func (s *PrefixStorage) fullPath(name string) string {
	return fmt.Sprintf("%s%s", s.prefix, name)
}

func NewPrefixStorage(prefix string, parentStorage logical.Storage) *PrefixStorage {
	return &PrefixStorage{
		prefix:        prefix,
		parentStorage: parentStorage,
	}
}

func NewPrefixStorageRequestFactory(prefix string) PrefixStorageRequestFactory {
	return func(request *logical.Request) *PrefixStorage {
		return NewPrefixStorage(prefix, request.Storage)
	}
}
