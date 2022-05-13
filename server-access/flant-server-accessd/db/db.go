package db

import (
	"context"

	"github.com/flant/negentropy/server-access/flant-server-accessd/types"
)

type UserDatabase interface {
	Migrate() error
	Sync(ctx context.Context, uwg types.UsersWithGroups) error
	GetChanges(ctx context.Context, uwg types.UsersWithGroups) ([]types.User, []types.User, error)
	Close()
}
