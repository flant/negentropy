/*
1. Создаём каркас: group, shadow, passwd. Из shadow FK на passwd.
2. Итерируем по пользователям, создаём домашние директории
3.
*/

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/go_bindata"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	dberrors "github.com/flant/server-access/flant-server-accessd/db/errors"
	"github.com/flant/server-access/flant-server-accessd/db/sqlite/migrations"
	"github.com/flant/server-access/flant-server-accessd/types"
)

const (
	currentDatabaseVersion          = 1
	usersTempTableCreateStatementv1 = `
CREATE TEMPORARY TABLE temp_users
(
    name        TEXT PRIMARY KEY                NOT NULL,
    uid         INTEGER NOT NULL,
    gid         INTEGER             NOT NULL,
    gecos       TEXT,
    homedir     TEXT                NOT NULL,
    shell       TEXT                NOT NULL,
    hashed_pass TEXT                NOT NULL
);`
	oldUsersStatementv1 = `
SELECT name, uid, gid, gecos, homedir, shell, hashed_pass FROM users
WHERE (name, uid, gid, homedir) IN 
(
	SELECT name, uid, gid, homedir FROM users
	EXCEPT
	SELECT name, uid, gid, homedir FROM temp.temp_users
);
`
	newUsersStatementv1 = `
SELECT name, uid, gid, gecos, homedir, shell, hashed_pass FROM temp.temp_users
WHERE (name, uid, gid, homedir) IN 
(
	SELECT name, uid, gid, homedir FROM temp.temp_users
	EXCEPT
	SELECT name, uid, gid, homedir FROM users
)
`
)

type UserDatabase struct {
	db *sqlx.DB
}

func NewUserDatabase(path string, ro ...bool) (*UserDatabase, error) {
	var roFlag bool
	if len(ro) > 0 && ro[0] {
		roFlag = true
	}

	dsn := path
	if roFlag {
		dsn += "?mode=ro"
	}

	database, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return &UserDatabase{db: database}, nil
}

func (db *UserDatabase) Close() {
	if db.db != nil {
		db.db.Close()
	}
}

func (db *UserDatabase) Migrate() error {
	s := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})

	sourceDriver, err := bindata.WithInstance(s)
	if err != nil {
		return err
	}

	dbDriver, err := sqlite3.WithInstance(db.db.DB, &sqlite3.Config{})
	if err != nil {
		return err
	}

	migrator, err := migrate.NewWithInstance("go-bindata", sourceDriver, "user_database", dbDriver)
	if err != nil {
		return err
	}

	err = migrator.Migrate(currentDatabaseVersion)
	if err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	}

	return nil
}

func (db *UserDatabase) Sync(ctx context.Context, uwg types.UsersWithGroups) error {
	tx, err := db.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	_, err = tx.Exec(`DELETE FROM users`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM groups`)
	if err != nil {
		return err
	}

	preparedInsertUsers, err := tx.PrepareNamed(`
INSERT INTO users (name, uid, gid, gecos, homedir, shell, hashed_pass) 
VALUES (:name, :uid, :gid, :gecos, :homedir, :shell, :hashed_pass);`)
	if err != nil {
		return err
	}

	for _, user := range uwg.Users {
		_, err := preparedInsertUsers.Exec(user)
		if err != nil {
			return err
		}
	}

	preparedInsertGroups, err := tx.PrepareNamed(`
INSERT INTO groups (name, gid) 
VALUES (:name, :gid);`)
	if err != nil {
		return err
	}

	for _, group := range uwg.Groups {
		_, err := preparedInsertGroups.Exec(group)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (db *UserDatabase) GetChanges(ctx context.Context, uwg types.UsersWithGroups) ([]types.User, []types.User, error) {
	var (
		newUsers []types.User
		oldUsers []types.User
	)

	tx, err := db.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	_, err = tx.Exec(usersTempTableCreateStatementv1)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create TEMP users table: %s", err)
	}

	preparedInsert, err := tx.PrepareNamed(`
INSERT INTO temp.temp_users (name, uid, gid, gecos, homedir, shell, hashed_pass) 
VALUES (:name, :uid, :gid, :gecos, :homedir, :shell, :hashed_pass);`)
	if err != nil {
		return nil, nil, err
	}

	for _, user := range uwg.Users {
		_, err := preparedInsert.Exec(user)
		if err != nil {
			return nil, nil, err
		}
	}

	err = tx.Select(&newUsers, newUsersStatementv1)
	if err != nil {
		return nil, nil, err
	}

	err = tx.Select(&oldUsers, oldUsersStatementv1)
	if err != nil {
		return nil, nil, err
	}

	_, err = tx.Exec(`DROP TABLE IF EXISTS temp.temp_users`)
	if err != nil {
		return nil, nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, nil, err
	}

	return newUsers, oldUsers, nil
}

func (db *UserDatabase) GetUserByName(ctx context.Context, name string) (types.User, error) {
	var user types.User

	row := db.db.QueryRowxContext(ctx, `SELECT * FROM users WHERE name == ?`, name)
	if row.Err() != nil {
		return types.User{}, row.Err()
	}

	err := row.StructScan(&user)
	if errors.Is(err, sql.ErrNoRows) {
		return types.User{}, dberrors.NewEntryNotFound(fmt.Sprintf("no entries for User name %q", name))
	} else if err != nil {
		return types.User{}, err
	}

	return user, err
}

func (db *UserDatabase) GetUserByUID(ctx context.Context, uid uint) (types.User, error) {
	var user types.User

	row := db.db.QueryRowxContext(ctx, `SELECT * FROM users WHERE uid == ?`, uid)
	if row.Err() != nil {
		return types.User{}, row.Err()
	}

	err := row.StructScan(&user)
	if errors.Is(err, sql.ErrNoRows) {
		return types.User{}, dberrors.NewEntryNotFound(fmt.Sprintf("no entries for User UID %q", uid))
	} else if err != nil {
		return types.User{}, err
	}

	return user, err
}

func (db *UserDatabase) GetUsers(ctx context.Context) ([]types.User, error) {
	var users []types.User
	err := db.db.SelectContext(ctx, &users, `SELECT * FROM users;`)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (db *UserDatabase) GetGroupByName(ctx context.Context, name string) (types.Group, error) {
	var group types.Group

	row := db.db.QueryRowxContext(ctx, `SELECT * FROM groups WHERE name == ?`, name)
	if row.Err() != nil {
		return types.Group{}, row.Err()
	}

	err := row.StructScan(&group)
	if errors.Is(err, sql.ErrNoRows) {
		return types.Group{}, dberrors.NewEntryNotFound(fmt.Sprintf("no entries for Group name %q", name))
	} else if err != nil {
		return types.Group{}, err
	}

	return group, err
}

func (db *UserDatabase) GetGroupByGID(ctx context.Context, gid uint) (types.Group, error) {
	var group types.Group

	row := db.db.QueryRowxContext(ctx, `SELECT * FROM groups WHERE gid == ?`, gid)
	if row.Err() != nil {
		return types.Group{}, row.Err()
	}

	err := row.StructScan(&group)
	if errors.Is(err, sql.ErrNoRows) {
		return types.Group{}, dberrors.NewEntryNotFound(fmt.Sprintf("no entries for Group GID %q", gid))
	} else if err != nil {
		return types.Group{}, err
	}

	return group, err
}

func (db *UserDatabase) GetGroups(ctx context.Context) ([]types.Group, error) {
	var groups []types.Group
	err := db.db.SelectContext(ctx, &groups, `SELECT * FROM groups;`)
	if err != nil {
		return nil, err
	}

	return groups, nil
}
