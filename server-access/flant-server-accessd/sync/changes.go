package sync

import (
	"context"

	"github.com/flant/server-access/flant-server-accessd/db"
	"github.com/flant/server-access/flant-server-accessd/system"
	"github.com/flant/server-access/flant-server-accessd/types"
)

func ApplyChanges(ctx context.Context, db db.UserDatabase, uwg types.UsersWithGroups) error {
	newUsers, oldUsers, err := db.GetChanges(ctx, uwg)
	if err != nil {
		return err
	}

	var sysOp system.Interface
	sysOp = system.NewSystemOperator()

	for _, oldUser := range oldUsers {
		err := sysOp.DeleteHomeDir(oldUser.HomeDir)
		if err != nil {
			return err
		}

		err = sysOp.PurgeUserLegacy(oldUser.Name)
		if err != nil {
			return err
		}
	}

	for _, newUser := range newUsers {
		err := sysOp.CreateHomeDir(newUser.HomeDir, int(newUser.Uid), int(newUser.Gid))
		if err != nil {
			return err
		}

		err = sysOp.CreateAuthorizedKeysFile(newUser.HomeDir, newUser.Principal)
		if err != nil {
			return err
		}

		err = sysOp.FixChown(newUser.HomeDir, int(newUser.Uid), int(newUser.Gid))
		if err != nil {
			return err
		}
	}

	return nil
}
