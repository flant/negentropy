package backend

import (
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
)

type Backend struct {
	memStorage *sharedio.MemoryStore
	deps *usecase.Depends
}

// DO NOT USE DIRECTLY, use token controller instead
func NewBackend(storage *sharedio.MemoryStore, deps *usecase.Depends) *Backend {
	return &Backend{
		memStorage: storage,
		deps: deps,
	}
}
