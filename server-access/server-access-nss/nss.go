package main

import (
	"context"
	"time"

	nss "github.com/protosam/go-libnss"
	"github.com/protosam/go-libnss/structs"

	dberrors "github.com/flant/server-access/flant-server-accessd/db/errors"
	"github.com/flant/server-access/flant-server-accessd/db/sqlite"
	"github.com/flant/server-access/flant-server-accessd/types"
)

type UserDatabase interface {
	GetUserByName(ctx context.Context, name string) (types.User, error)
	GetUserByUID(ctx context.Context, uid uint) (types.User, error)
	GetUsers(ctx context.Context) ([]types.User, error)
	GetGroupByName(ctx context.Context, name string) (types.Group, error)
	GetGroupByGID(ctx context.Context, gid uint) (types.Group, error)
	GetGroups(ctx context.Context) ([]types.Group, error)
}

var userDB UserDatabase

func main() {}

func init() {
	database, err := sqlite.NewUserDatabase("/home/zuzzas/sqlite_tests/server_access.db", true)
	if err == nil {
		userDB = database
	}

	nss.SetImpl(Provider{})
}

type Provider struct{ nss.LIBNSS }

func (p Provider) PasswdAll() (nss.Status, []structs.Passwd) {
	var passwds []structs.Passwd

	if !isDBValid() {
		return nss.StatusUnavail, passwds
	}

	ctx, cancel := defaultContext()
	defer cancel()

	users, err := userDB.GetUsers(ctx)
	if err != nil {
		return nss.StatusUnavail, passwds
	}

	for _, user := range users {
		passwds = append(passwds, structs.Passwd{
			Username: user.Name,
			Password: "x",
			UID:      user.Uid,
			GID:      user.Gid,
			Gecos:    user.Gecos,
			Dir:      user.HomeDir,
			Shell:    user.Shell,
		})
	}

	return nss.StatusSuccess, passwds
}

func (p Provider) PasswdByName(name string) (nss.Status, structs.Passwd) {
	if !isDBValid() {
		return nss.StatusUnavail, structs.Passwd{}
	}

	ctx, cancel := defaultContext()
	defer cancel()

	user, err := userDB.GetUserByName(ctx, name)
	if dberrors.IsEntryNotFound(err) {
		return nss.StatusNotfound, structs.Passwd{}
	} else if err != nil {
		return nss.StatusUnavail, structs.Passwd{}
	}

	return nss.StatusSuccess, structs.Passwd{
		Username: user.Name,
		Password: "x",
		UID:      user.Uid,
		GID:      user.Gid,
		Gecos:    user.Gecos,
		Dir:      user.HomeDir,
		Shell:    user.Shell,
	}
}

func (p Provider) PasswdByUid(uid uint) (nss.Status, structs.Passwd) {
	if !isDBValid() {
		return nss.StatusUnavail, structs.Passwd{}
	}

	ctx, cancel := defaultContext()
	defer cancel()

	user, err := userDB.GetUserByUID(ctx, uid)
	if dberrors.IsEntryNotFound(err) {
		return nss.StatusNotfound, structs.Passwd{}
	} else if err != nil {
		return nss.StatusUnavail, structs.Passwd{}
	}

	return nss.StatusSuccess, structs.Passwd{
		Username: user.Name,
		Password: "x",
		UID:      user.Uid,
		GID:      user.Gid,
		Gecos:    user.Gecos,
		Dir:      user.HomeDir,
		Shell:    user.Shell,
	}
}

func (p Provider) GroupAll() (nss.Status, []structs.Group) {
	var groups []structs.Group

	if !isDBValid() {
		return nss.StatusUnavail, groups
	}

	ctx, cancel := defaultContext()
	defer cancel()

	grps, err := userDB.GetGroups(ctx)
	if err != nil {
		return nss.StatusUnavail, groups
	}

	for _, group := range grps {
		groups = append(groups, structs.Group{
			Groupname: group.Name,
			Password:  "x",
			GID:       group.Gid,
		})
	}

	return nss.StatusSuccess, groups
}

func (p Provider) GroupByName(name string) (nss.Status, structs.Group) {
	if !isDBValid() {
		return nss.StatusUnavail, structs.Group{}
	}

	ctx, cancel := defaultContext()
	defer cancel()

	grp, err := userDB.GetGroupByName(ctx, name)
	if dberrors.IsEntryNotFound(err) {
		return nss.StatusNotfound, structs.Group{}
	} else if err != nil {
		return nss.StatusUnavail, structs.Group{}
	}

	return nss.StatusSuccess, structs.Group{
		Groupname: grp.Name,
		Password:  "x",
		GID:       grp.Gid,
	}
}

func (p Provider) GroupByGid(gid uint) (nss.Status, structs.Group) {
	if !isDBValid() {
		return nss.StatusUnavail, structs.Group{}
	}

	ctx, cancel := defaultContext()
	defer cancel()

	grp, err := userDB.GetGroupByGID(ctx, gid)
	if dberrors.IsEntryNotFound(err) {
		return nss.StatusNotfound, structs.Group{}
	} else if err != nil {
		return nss.StatusUnavail, structs.Group{}
	}

	return nss.StatusSuccess, structs.Group{
		Groupname: grp.Name,
		Password:  "x",
		GID:       grp.Gid,
	}
}

func (p Provider) ShadowAll() (nss.Status, []structs.Shadow) {
	var shadows []structs.Shadow

	if !isDBValid() {
		return nss.StatusUnavail, shadows
	}

	ctx, cancel := defaultContext()
	defer cancel()

	users, err := userDB.GetUsers(ctx)
	if err != nil {
		return nss.StatusUnavail, shadows
	}

	for _, user := range users {
		shadows = append(shadows, structs.Shadow{
			Username:        user.Name,
			Password:        user.HashedPass,
			LastChange:      -1,
			MinChange:       -1,
			MaxChange:       -1,
			PasswordWarn:    -1,
			InactiveLockout: -1,
			ExpirationDate:  -1,
			Reserved:        -1,
		})
	}

	return nss.StatusSuccess, shadows
}

func (p Provider) ShadowByName(name string) (nss.Status, structs.Shadow) {
	if !isDBValid() {
		return nss.StatusUnavail, structs.Shadow{}
	}

	ctx, cancel := defaultContext()
	defer cancel()

	user, err := userDB.GetUserByName(ctx, name)
	if dberrors.IsEntryNotFound(err) {
		return nss.StatusNotfound, structs.Shadow{}
	} else if err != nil {
		return nss.StatusUnavail, structs.Shadow{}
	}

	return nss.StatusSuccess, structs.Shadow{
		Username:        user.Name,
		Password:        user.HashedPass,
		LastChange:      -1,
		MinChange:       -1,
		MaxChange:       -1,
		PasswordWarn:    -1,
		InactiveLockout: -1,
		ExpirationDate:  -1,
		Reserved:        -1,
	}
}

func defaultContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func isDBValid() bool {
	return userDB != nil
}
