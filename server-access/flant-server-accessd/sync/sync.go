package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/flant/server-access/flant-server-accessd/db"
	"github.com/flant/server-access/flant-server-accessd/types"
	"github.com/flant/server-access/vault"
)

type Periodic struct {
	DB            db.UserDatabase
	AuthdSettings vault.AuthdSettings
	Settings      vault.ServerAccessSettings
}

func NewPeriodic(dbInstance db.UserDatabase, authdSettings vault.AuthdSettings, settings vault.ServerAccessSettings) *Periodic {
	return &Periodic{
		DB:            dbInstance,
		AuthdSettings: authdSettings,
		Settings:      settings,
	}
}

func (p *Periodic) Start() {
	go func() {
		for {
			err := p.SyncUsers()
			if err != nil {
				os.Exit(1)
			}
			time.Sleep(30 * time.Second)
		}
	}()
}

func (p *Periodic) SyncUsers() error {
	vaultClient, err := vault.ClientFromAuthd(p.AuthdSettings)
	if err != nil {
		log.Printf("Open Vault session: %v", err)
		return nil
	}

	posixUsers, err := vault.NewFlantIAMAuth(vaultClient).PosixUsers(p.Settings)
	if err != nil {
		log.Printf("List POSIX users: %v", err)
		return nil
	}

	// TODO: Can we use PosixUsers directly without conversion?
	uwg, err := ConvertPOSIXUsers(posixUsers)
	if err != nil {
		log.Printf("Convert POSIX users: %v", err)
		return nil
	}

	err = ApplyChanges(context.Background(), p.DB, uwg)
	if err != nil {
		return fmt.Errorf("apply user changes: %v", err)
	}

	err = p.DB.Sync(context.Background(), uwg)
	if err != nil {
		return fmt.Errorf("sync database after apply user changes: %v", err)
	}

	return nil
}

func ConvertPOSIXUsers(posixUsers []vault.PosixUser) (types.UsersWithGroups, error) {
	groupMap := make(map[int]types.Group)
	userMap := make(map[int]types.User)

	for _, posixUser := range posixUsers {
		if _, has := userMap[posixUser.UID]; !has {
			userMap[posixUser.UID] = types.User{
				Name:       posixUser.Name,
				Uid:        uint(posixUser.UID),
				Gid:        uint(posixUser.Gid),
				Gecos:      posixUser.Gecos,
				HomeDir:    posixUser.HomeDir,
				Shell:      posixUser.Shell,
				HashedPass: posixUser.Password,
			}
		}

		if _, has := groupMap[posixUser.Gid]; !has {
			groupMap[posixUser.Gid] = types.Group{
				Name: fmt.Sprintf("srvgroup%d", posixUser.Gid),
				Gid:  uint(posixUser.Gid),
			}
		}
	}

	users := make([]types.User, 0)
	for _, user := range userMap {
		users = append(users, user)
	}
	groups := make([]types.Group, 0)
	for _, group := range groupMap {
		groups = append(groups, group)
	}

	return types.UsersWithGroups{Users: users, Groups: groups}, nil
}
