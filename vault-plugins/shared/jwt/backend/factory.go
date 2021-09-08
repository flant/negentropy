package backend

import (
	"fmt"

	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
)

type Backend struct {
	memStorage *sharedio.MemoryStore
	deps       *usecase.Depends
}

// DO NOT USE DIRECTLY, use token controller instead
func NewBackend(storage *sharedio.MemoryStore, deps *usecase.Depends) *Backend {
	return &Backend{
		memStorage: storage,
		deps:       deps,
	}
}

func (b *Backend) mustEnabled(txn *sharedio.MemoryStoreTxn) error {
	r, err := b.deps.StateRepo(txn)
	if err != nil {
		return err
	}

	enabled, err := r.IsEnabled()
	if err != nil {
		return err
	}

	if !enabled {
		return fmt.Errorf("jwt is not enabled")
	}

	return nil
}
