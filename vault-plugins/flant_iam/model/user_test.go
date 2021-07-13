package model

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	userUUID1 = "00000000-0000-0000-0000-000000000001"
	userUUID2 = "00000000-0000-0000-0000-000000000002"
	userUUID3 = "00000000-0000-0000-0000-000000000003"
	userUUID4 = "00000000-0000-0000-0000-000000000004"
	userUUID5 = "00000000-0000-0000-0000-000000000005"
)

var (
	user1 = User{
		UUID:           userUUID1,
		TenantUUID:     tenantUUID1,
		Identifier:     "user1",
		FullIdentifier: "user1@test",
		Email:          "user1@mail.com",
		Origin:         "test",
	}
	user2 = User{
		UUID:           userUUID2,
		TenantUUID:     tenantUUID1,
		Identifier:     "user2",
		FullIdentifier: "user2@test",
		Email:          "user2@mail.com",
		Origin:         "test",
	}
	user3 = User{
		UUID:           userUUID3,
		TenantUUID:     tenantUUID1,
		Identifier:     "user3",
		FullIdentifier: "user3@test",
		Email:          "user3@mail.com",
		Origin:         "test",
	}
	user4 = User{
		UUID:           userUUID4,
		TenantUUID:     tenantUUID1,
		Identifier:     "user4",
		FullIdentifier: "user4@test",
		Email:          "user4@mail.com",
		Origin:         "test",
	}
	user5 = User{
		UUID:           userUUID5,
		TenantUUID:     tenantUUID2,
		Identifier:     "user4",
		FullIdentifier: "user4@test",
		Email:          "user4@mail.com",
		Origin:         "test",
	}
)

func createUsers(t *testing.T, repo *UserRepository, users ...User) {
	for _, user := range users {
		tmp := user
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func userFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := NewUserRepository(tx)
	createUsers(t, repo, []User{user1, user2, user3, user4, user5}...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_UserDbSchema(t *testing.T) {
	schema := UserSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("user schema is invalid: %v", err)
	}
}

func Test_UserList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture).Txn(true)
	repo := NewUserRepository(tx)

	users, err := repo.List(tenantUUID1)

	dieOnErr(t, err)
	ids := make([]string, 0)
	for _, obj := range users {
		ids = append(ids, obj.ObjId())
	}
	checkDeepEqual(t, []string{userUUID1, userUUID2, userUUID3, userUUID4}, ids)
}
