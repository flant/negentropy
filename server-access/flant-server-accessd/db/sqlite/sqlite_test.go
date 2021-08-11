package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	dberrors "github.com/flant/server-access/flant-server-accessd/db/errors"
	"github.com/flant/server-access/flant-server-accessd/types"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dbFileName = "sqlite_test.db"
)

func getDeadlineContext(t *testing.T) (context.Context, func()) {
	t.Helper()

	deadline, ok := t.Deadline()
	if ok {
		return context.WithDeadline(context.Background(), deadline)
	}

	return context.Background(), func() {}
}

func newTestDB(t *testing.T) *UserDatabase {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	t.Cleanup(func() {
		err := os.Remove(dbPath)
		require.Nil(t, err)
	})

	db, err := NewUserDatabase(dbPath)
	require.Nil(t, err, "failed to instantiate DB instance")

	err = db.Migrate()
	require.Nil(t, err, "failed to migrate DB instance")

	return db
}

func addTestUsersAndGroups(t *testing.T, ctx context.Context, userDB *UserDatabase, uwg types.UsersWithGroups) {
	t.Helper()

	tx, err := userDB.db.BeginTxx(ctx, nil)
	require.Nil(t, err, "failed to begin SQL transaction")
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	insertUsers, err := tx.PrepareNamed(`
INSERT INTO users (name, uid, gid, gecos, homedir, shell, hashed_pass, principal) 
VALUES (:name, :uid, :gid, :gecos, :homedir, :shell, :hashed_pass, :principal);`)
	require.Nil(t, err, "failed to prepare users SQL query")

	insertGroups, err := tx.PrepareNamed(`
INSERT INTO groups (name, gid) 
VALUES (:name, :gid);`)
	require.Nil(t, err, "failed to prepare users SQL query")

	for _, user := range uwg.Users {
		_, err := insertUsers.Exec(user)
		require.Nil(t, err, "failed to insert user %+v", user)
	}

	for _, group := range uwg.Groups {
		_, err := insertGroups.Exec(group)
		require.Nil(t, err, "failed to insert group %+v", group)
	}

	err = tx.Commit()
	require.Nil(t, err, "failed to commit SQL transaction")
}

func TestUserDatabase_GetChanges(t *testing.T) {
	ctx, cancel := getDeadlineContext(t)
	defer cancel()

	tests := []struct {
		name         string
		existingUwg  types.UsersWithGroups
		newUwg       types.UsersWithGroups
		wantNewUsers []types.User
		wantOldUsers []types.User
	}{
		{
			name: "new users, empty database",
			newUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name:       "test",
						Uid:        1001,
						Gid:        1001,
						Gecos:      "Test user",
						HomeDir:    "/home/test-user",
						Shell:      "/bin/bash",
						HashedPass: "test",
						Principal:  "principal1",
					},
					{
						Name:       "test2",
						Uid:        1002,
						Gid:        1002,
						Gecos:      "Test user 2",
						HomeDir:    "/home/test-user2",
						Shell:      "/bin/bash",
						HashedPass: "test2",
						Principal:  "principal2",
					},
				},
			},
			wantNewUsers: []types.User{
				{
					Name:       "test",
					Uid:        1001,
					Gid:        1001,
					Gecos:      "Test user",
					HomeDir:    "/home/test-user",
					Shell:      "/bin/bash",
					HashedPass: "test",
					Principal:  "principal1",
				},
				{
					Name:       "test2",
					Uid:        1002,
					Gid:        1002,
					Gecos:      "Test user 2",
					HomeDir:    "/home/test-user2",
					Shell:      "/bin/bash",
					HashedPass: "test2",
					Principal:  "principal2",
				},
			},
		},
		{
			name: "new users, existing database, changing only shell, should not generate new or old users",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name:       "test",
						Uid:        1001,
						Gid:        1001,
						Gecos:      "Test user",
						HomeDir:    "/home/test-user",
						Shell:      "/bin/bash",
						HashedPass: "test",
						Principal:  "principal1",
					},
					{
						Name:       "test2",
						Uid:        1002,
						Gid:        1002,
						Gecos:      "Test user 2",
						HomeDir:    "/home/test-user2",
						Shell:      "/bin/bash",
						HashedPass: "test2",
						Principal:  "principal2",
					},
				},
			},
			newUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name:       "test",
						Uid:        1001,
						Gid:        1001,
						Gecos:      "Test user",
						HomeDir:    "/home/test-user",
						Shell:      "/bin/zsh",
						HashedPass: "test",
						Principal:  "principal1",
					},
					{
						Name:       "test2",
						Uid:        1002,
						Gid:        1002,
						Gecos:      "Test user 2",
						HomeDir:    "/home/test-user2",
						Shell:      "/bin/zsh",
						HashedPass: "test2",
						Principal:  "principal2",
					},
				},
			},
		},
		{
			name: "new users, existing database, changing homedir, should generate new and old users",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name:       "test",
						Uid:        1001,
						Gid:        1001,
						Gecos:      "Test user",
						HomeDir:    "/home/test-user",
						Shell:      "/bin/bash",
						HashedPass: "test",
						Principal:  "principal1",
					},
					{
						Name:       "test2",
						Uid:        1002,
						Gid:        1002,
						Gecos:      "Test user 2",
						HomeDir:    "/home/test-user-2",
						Shell:      "/bin/bash",
						HashedPass: "test2",
						Principal:  "principal2",
					},
				},
			},
			newUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name:       "test",
						Uid:        1001,
						Gid:        1001,
						Gecos:      "Test user",
						HomeDir:    "/home/test-user-3",
						Shell:      "/bin/bash",
						HashedPass: "test",
						Principal:  "principal1",
					},
					{
						Name:       "test2",
						Uid:        1002,
						Gid:        1002,
						Gecos:      "Test user 2",
						HomeDir:    "/home/test-user-4",
						Shell:      "/bin/bash",
						HashedPass: "test2",
						Principal:  "principal2",
					},
				},
			},
			wantNewUsers: []types.User{
				{
					Name:       "test",
					Uid:        1001,
					Gid:        1001,
					Gecos:      "Test user",
					HomeDir:    "/home/test-user-3",
					Shell:      "/bin/bash",
					HashedPass: "test",
					Principal:  "principal1",
				},
				{
					Name:       "test2",
					Uid:        1002,
					Gid:        1002,
					Gecos:      "Test user 2",
					HomeDir:    "/home/test-user-4",
					Shell:      "/bin/bash",
					HashedPass: "test2",
					Principal:  "principal2",
				},
			},
			wantOldUsers: []types.User{
				{
					Name:       "test",
					Uid:        1001,
					Gid:        1001,
					Gecos:      "Test user",
					HomeDir:    "/home/test-user",
					Shell:      "/bin/bash",
					HashedPass: "test",
					Principal:  "principal1",
				},
				{
					Name:       "test2",
					Uid:        1002,
					Gid:        1002,
					Gecos:      "Test user 2",
					HomeDir:    "/home/test-user-2",
					Shell:      "/bin/bash",
					HashedPass: "test2",
					Principal:  "principal2",
				},
			},
		},
		{
			name: "removing one user, existing database, should generate only old users",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name:       "test",
						Uid:        1001,
						Gid:        1001,
						Gecos:      "Test user",
						HomeDir:    "/home/test-user",
						Shell:      "/bin/bash",
						HashedPass: "test",
						Principal:  "principal1",
					},
					{
						Name:       "test2",
						Uid:        1002,
						Gid:        1002,
						Gecos:      "Test user 2",
						HomeDir:    "/home/test-user-2",
						Shell:      "/bin/bash",
						HashedPass: "test2",
						Principal:  "principal2",
					},
				},
			},
			newUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name:       "test",
						Uid:        1001,
						Gid:        1001,
						Gecos:      "Test user",
						HomeDir:    "/home/test-user",
						Shell:      "/bin/bash",
						HashedPass: "test",
						Principal:  "principal1",
					},
				},
			},
			wantOldUsers: []types.User{
				{
					Name:       "test2",
					Uid:        1002,
					Gid:        1002,
					Gecos:      "Test user 2",
					HomeDir:    "/home/test-user-2",
					Shell:      "/bin/bash",
					HashedPass: "test2",
					Principal:  "principal2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			addTestUsersAndGroups(t, ctx, db, tt.existingUwg)

			newUsers, oldUsers, err := db.GetChanges(ctx, tt.newUwg)
			require.Nil(t, err)

			assert.Equal(t, tt.wantNewUsers, newUsers)
			assert.Equal(t, tt.wantOldUsers, oldUsers)
		})
	}
}

func TestUserDatabase_GetGroupByGID(t *testing.T) {
	ctx, cancel := getDeadlineContext(t)
	defer cancel()

	tests := []struct {
		name                 string
		existingUwg          types.UsersWithGroups
		gid                  uint
		wantGroup            types.Group
		wantEntryNotFoundErr bool
	}{
		{
			name:                 "getting no groups from empty database",
			gid:                  1001,
			wantEntryNotFoundErr: true,
		},
		{
			name: "getting one group from database with one entry",
			existingUwg: types.UsersWithGroups{
				Groups: []types.Group{
					{
						Name: "test",
						Gid:  1001,
					},
				},
			},
			gid: 1001,
			wantGroup: types.Group{
				Name: "test",
				Gid:  1001,
			},
		},
		{
			name: "getting one group from database with two entries",
			existingUwg: types.UsersWithGroups{
				Groups: []types.Group{
					{
						Name: "test",
						Gid:  1001,
					},
					{
						Name: "test2",
						Gid:  1002,
					},
				},
			},
			gid: 1001,
			wantGroup: types.Group{
				Name: "test",
				Gid:  1001,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			addTestUsersAndGroups(t, ctx, db, tt.existingUwg)

			group, err := db.GetGroupByGID(ctx, tt.gid)
			if tt.wantEntryNotFoundErr {
				require.True(t, dberrors.IsEntryNotFound(err))
			} else {
				require.Nil(t, err)
			}

			assert.Equal(t, tt.wantGroup, group)
		})
	}
}

func TestUserDatabase_GetGroupByName(t *testing.T) {
	ctx, cancel := getDeadlineContext(t)
	defer cancel()

	tests := []struct {
		name                 string
		existingUwg          types.UsersWithGroups
		groupName            string
		wantGroup            types.Group
		wantEntryNotFoundErr bool
	}{
		{
			name:                 "getting no groups from empty database",
			groupName:            "test",
			wantEntryNotFoundErr: true,
		},
		{
			name: "getting one group from database with one entry",
			existingUwg: types.UsersWithGroups{
				Groups: []types.Group{
					{
						Name: "test",
						Gid:  1001,
					},
				},
			},
			groupName: "test",
			wantGroup: types.Group{
				Name: "test",
				Gid:  1001,
			},
		},
		{
			name: "getting one group from database with two entries",
			existingUwg: types.UsersWithGroups{
				Groups: []types.Group{
					{
						Name: "test",
						Gid:  1001,
					},
					{
						Name: "test2",
						Gid:  1002,
					},
				},
			},
			groupName: "test",
			wantGroup: types.Group{
				Name: "test",
				Gid:  1001,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			addTestUsersAndGroups(t, ctx, db, tt.existingUwg)

			group, err := db.GetGroupByName(ctx, tt.groupName)
			if tt.wantEntryNotFoundErr {
				require.True(t, dberrors.IsEntryNotFound(err))
			} else {
				require.Nil(t, err)
			}

			assert.Equal(t, tt.wantGroup, group)
		})
	}
}

func TestUserDatabase_GetGroups(t *testing.T) {
	ctx, cancel := getDeadlineContext(t)
	defer cancel()

	tests := []struct {
		name        string
		existingUwg types.UsersWithGroups
		wantGroups  []types.Group
	}{
		{
			name: "getting no groups from empty database",
		},
		{
			name: "getting one group from database with one entry",
			existingUwg: types.UsersWithGroups{
				Groups: []types.Group{
					{
						Name: "test",
						Gid:  1001,
					},
				},
			},
			wantGroups: []types.Group{
				{
					Name: "test",
					Gid:  1001,
				},
			},
		},
		{
			name: "getting two groups from database with two entries",
			existingUwg: types.UsersWithGroups{
				Groups: []types.Group{
					{
						Name: "test",
						Gid:  1001,
					},
					{
						Name: "test2",
						Gid:  1002,
					},
				},
			},
			wantGroups: []types.Group{
				{
					Name: "test",
					Gid:  1001,
				},
				{
					Name: "test2",
					Gid:  1002,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			addTestUsersAndGroups(t, ctx, db, tt.existingUwg)

			groups, err := db.GetGroups(ctx)
			require.Nil(t, err)

			assert.Equal(t, tt.wantGroups, groups)
		})
	}
}

func TestUserDatabase_GetUserByName(t *testing.T) {
	ctx, cancel := getDeadlineContext(t)
	defer cancel()

	tests := []struct {
		name                 string
		existingUwg          types.UsersWithGroups
		userName             string
		wantUser             types.User
		wantEntryNotFoundErr bool
	}{
		{
			name:                 "getting no users from empty database",
			userName:             "test",
			wantEntryNotFoundErr: true,
		},
		{
			name: "getting one user from database with one entry",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name: "test",
						Uid:  1001,
					},
				},
			},
			userName: "test",
			wantUser: types.User{
				Name: "test",
				Uid:  1001,
			},
		},
		{
			name: "getting one user from database with two entries",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name: "test",
						Uid:  1001,
					},
					{
						Name: "test2",
						Uid:  1002,
					},
				},
			},
			userName: "test",
			wantUser: types.User{
				Name: "test",
				Uid:  1001,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			addTestUsersAndGroups(t, ctx, db, tt.existingUwg)

			user, err := db.GetUserByName(ctx, tt.userName)
			if tt.wantEntryNotFoundErr {
				require.True(t, dberrors.IsEntryNotFound(err))
			} else {
				require.Nil(t, err)
			}

			assert.Equal(t, tt.wantUser, user)
		})
	}
}

func TestUserDatabase_GetUserByUID(t *testing.T) {
	ctx, cancel := getDeadlineContext(t)
	defer cancel()

	tests := []struct {
		name                 string
		existingUwg          types.UsersWithGroups
		uid                  uint
		wantUser             types.User
		wantEntryNotFoundErr bool
	}{
		{
			name:                 "getting no users from empty database",
			uid:                  1001,
			wantEntryNotFoundErr: true,
		},
		{
			name: "getting one user from database with one entry",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name: "test",
						Uid:  1001,
					},
				},
			},
			uid: 1001,
			wantUser: types.User{
				Name: "test",
				Uid:  1001,
			},
		},
		{
			name: "getting one user from database with two entries",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name: "test",
						Uid:  1001,
					},
					{
						Name: "test2",
						Uid:  1002,
					},
				},
			},
			uid: 1001,
			wantUser: types.User{
				Name: "test",
				Uid:  1001,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			addTestUsersAndGroups(t, ctx, db, tt.existingUwg)

			user, err := db.GetUserByUID(ctx, tt.uid)
			if tt.wantEntryNotFoundErr {
				require.True(t, dberrors.IsEntryNotFound(err))
			} else {
				require.Nil(t, err)
			}

			assert.Equal(t, tt.wantUser, user)
		})
	}
}

func TestUserDatabase_GetUsers(t *testing.T) {
	ctx, cancel := getDeadlineContext(t)
	defer cancel()

	tests := []struct {
		name        string
		existingUwg types.UsersWithGroups
		wantUsers   []types.User
	}{
		{
			name: "getting no users from empty database",
		},
		{
			name: "getting one user from database with one entry",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name: "test",
						Uid:  1001,
					},
				},
			},
			wantUsers: []types.User{
				{
					Name: "test",
					Uid:  1001,
				},
			},
		},
		{
			name: "getting two users from database with two entries",
			existingUwg: types.UsersWithGroups{
				Users: []types.User{
					{
						Name: "test",
						Uid:  1001,
					},
					{
						Name: "test2",
						Uid:  1002,
					},
				},
			},
			wantUsers: []types.User{
				{
					Name: "test",
					Uid:  1001,
				},
				{
					Name: "test2",
					Uid:  1002,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			addTestUsersAndGroups(t, ctx, db, tt.existingUwg)

			users, err := db.GetUsers(ctx)
			require.Nil(t, err)

			assert.Equal(t, tt.wantUsers, users)
		})
	}
}

func TestUserDatabase_Sync(t *testing.T) {
	type fields struct {
		db *sqlx.DB
	}
	type args struct {
		ctx context.Context
		uwg types.UsersWithGroups
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &UserDatabase{
				db: tt.fields.db,
			}
			if err := db.Sync(tt.args.ctx, tt.args.uwg); (err != nil) != tt.wantErr {
				t.Errorf("Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
