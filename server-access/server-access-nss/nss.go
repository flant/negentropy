package main

import (
	"context"
	"fmt"
	"os"
	"time"

	nss "github.com/protosam/go-libnss"
	"github.com/protosam/go-libnss/structs"

	dberrors "github.com/flant/server-access/flant-server-accessd/db/errors"
	"github.com/flant/server-access/flant-server-accessd/db/sqlite"
	"github.com/flant/server-access/flant-server-accessd/types"
)

type UserDatabase interface {
	Close()
	GetUserByName(ctx context.Context, name string) (types.User, error)
	GetUserByUID(ctx context.Context, uid uint) (types.User, error)
	GetUsers(ctx context.Context) ([]types.User, error)
	GetGroupByName(ctx context.Context, name string) (types.Group, error)
	GetGroupByGID(ctx context.Context, gid uint) (types.Group, error)
	GetGroups(ctx context.Context) ([]types.Group, error)
}

func main() {}

func init() {
	nss.SetImpl(Provider{})
}

var UserDatabasePath = "/opt/negentropy/server-access.db"

func OpenUserDatabase() (UserDatabase, error) {
	return sqlite.NewUserDatabase(UserDatabasePath, true)
}

type Provider struct{ nss.LIBNSS }

func (p Provider) PasswdAll() (nss.Status, []structs.Passwd) {
	log := NewNssLogger()
	defer log.Close()
	log.Debugf("PasswdAll\n")

	var passwds []structs.Passwd

	userDB, err := OpenUserDatabase()
	if err != nil {
		return nss.StatusUnavail, passwds
	}
	defer userDB.Close()

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
	log := NewNssLogger()
	defer log.Close()
	log.Debugf("PasswdByName %s\n", name)

	userDB, err := OpenUserDatabase()
	if err != nil {
		return nss.StatusUnavail, structs.Passwd{}
	}
	defer userDB.Close()

	ctx, cancel := defaultContext()
	defer cancel()

	log.Debugf("PasswdByName get user by name\n")

	user, err := userDB.GetUserByName(ctx, name)
	if dberrors.IsEntryNotFound(err) {
		log.Debugf("PasswdByName not found\n")
		return nss.StatusNotfound, structs.Passwd{}
	} else if err != nil {
		log.Debugf("PasswdByName err: %v\n", err)
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
	log := NewNssLogger()
	defer log.Close()

	log.Debugf("PasswdByUid %d\n", uid)

	userDB, err := OpenUserDatabase()
	if err != nil {
		return nss.StatusUnavail, structs.Passwd{}
	}
	defer userDB.Close()

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
	log := NewNssLogger()
	defer log.Close()

	log.Debugf("GroupAll\n")

	var groups []structs.Group

	userDB, err := OpenUserDatabase()
	if err != nil {
		return nss.StatusUnavail, groups
	}
	defer userDB.Close()

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
	log := NewNssLogger()
	defer log.Close()

	log.Debugf("GroupByName %d\n", name)

	userDB, err := OpenUserDatabase()
	if err != nil {
		return nss.StatusUnavail, structs.Group{}
	}
	defer userDB.Close()

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
	log := NewNssLogger()
	defer log.Close()

	log.Debugf("GroupByGid %d\n", gid)

	userDB, err := OpenUserDatabase()
	if err != nil {
		return nss.StatusUnavail, structs.Group{}
	}
	defer userDB.Close()

	ctx, cancel := defaultContext()
	defer cancel()

	grp, err := userDB.GetGroupByGID(ctx, gid)
	if dberrors.IsEntryNotFound(err) {
		log.Debugf("GroupByGid not found: %v\n", err)
		return nss.StatusNotfound, structs.Group{}
	} else if err != nil {
		log.Debugf("GroupByGid err: %v\n", err)
		return nss.StatusUnavail, structs.Group{}
	}

	return nss.StatusSuccess, structs.Group{
		Groupname: grp.Name,
		Password:  "x",
		GID:       grp.Gid,
	}
}

func (p Provider) ShadowAll() (nss.Status, []structs.Shadow) {
	log := NewNssLogger()
	defer log.Close()

	log.Debugf("ShadowAll\n")

	var shadows []structs.Shadow

	userDB, err := OpenUserDatabase()
	if err != nil {
		return nss.StatusUnavail, shadows
	}
	defer userDB.Close()

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
	log := NewNssLogger()
	defer log.Close()

	log.Debugf("ShadowByName %s\n", name)

	userDB, err := OpenUserDatabase()
	if err != nil {
		return nss.StatusUnavail, structs.Shadow{}
	}
	defer userDB.Close()

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

const DefaultLoggerFile = "./libnss-flantauth.log"

type NssLogger struct {
	enabled bool
	f       *os.File
}

func NewNssLogger() *NssLogger {
	l := &NssLogger{}
	if os.Getenv("FLANTAUTH_DEBUG") == "yes" {
		l.enabled = true
		f, err := os.Create(DefaultLoggerFile)
		if err != nil {
			l.enabled = false
		} else {
			l.f = f
		}
	}
	return l
}

func (l *NssLogger) Close() {
	if l.f != nil {
		l.f.Close()
	}
}

func (l *NssLogger) Printf(format string, a ...interface{}) {
	if l.enabled {
		_, _ = fmt.Fprintf(l.f, format, a...)
	}
}

func (l *NssLogger) Debugf(format string, a ...interface{}) {
	l.Printf(format, a...)
}
