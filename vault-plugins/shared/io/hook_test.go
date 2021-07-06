package io

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHooks(t *testing.T) {
	t.Run("test insert", func(t *testing.T) {
		mem, err := NewMemoryStore(testSchema(), nil)
		require.NoError(t, err)

		hook := ObjectHook{
			Events:  []HookEvent{HookEventInsert},
			ObjType: "group",
			CallbackFn: func(txn *MemoryStoreTxn, event HookEvent, obj interface{}) error {
				if event != HookEventInsert {
					return nil
				}

				group := obj.(*group)

				userIter, err := txn.Get("user", "id")
				if err != nil {
					return err
				}
				for {
					raw := userIter.Next()
					if raw == nil {
						break
					}
					user := raw.(*user)
					for _, us := range group.Users {
						if us == user.UUID {
							user.Groups = append(user.Groups, group.Name)
							err = txn.Insert("user", user)
							if err != nil {
								return err
							}
							break
						}
					}

				}
				return nil
			},
		}

		mem.RegisterHook(hook)

		user1 := &user{UUID: "1"}
		user2 := &user{UUID: "2"}
		user3 := &user{UUID: "3"}
		txn := mem.Txn(true)
		err = txn.Insert("user", user1)
		assert.NoError(t, err)
		err = txn.Insert("user", user2)
		assert.NoError(t, err)
		err = txn.Insert("user", user3)
		assert.NoError(t, err)

		gr := &group{UUID: "777", Name: "super", Users: []string{"1", "2"}}
		err = txn.Insert("group", gr)
		assert.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)

		ntxn := mem.Txn(false)

		userIter, err := ntxn.Get("user", "id")
		if err != nil {
			t.Fatal(err)
		}
		for {
			raw := userIter.Next()
			if raw == nil {
				break
			}
			user := raw.(*user)
			switch user.UUID {
			case "1":
				require.Len(t, user.Groups, 1)
				assert.Equal(t, "super", user.Groups[0])

			case "2":
				require.Len(t, user.Groups, 1)
				assert.Equal(t, "super", user.Groups[0])

			case "3":
				require.Len(t, user.Groups, 0)
			}
		}
	})
}

type user struct {
	UUID   string   `json:"uuid"`
	Groups []string `json:"groups"`
}

func (u user) ObjType() string {
	return "user"
}
func (u user) ObjId() string {
	return u.UUID
}
func (u user) Marshal(_ bool) ([]byte, error) {
	return json.Marshal(u)
}

type group struct {
	UUID  string   `json:"uuid"`
	Name  string   `json:"name"`
	Users []string `json:"users"`
}

func (g group) ObjType() string {
	return "group"
}
func (g group) ObjId() string {
	return g.UUID
}
func (g group) Marshal(_ bool) ([]byte, error) {
	return json.Marshal(g)
}

func testSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"user": {
				Name: "user",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "UUID",
						},
					},
				},
			},
			"group": {
				Name: "group",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "UUID",
						},
					},
				},
			},
		},
	}
}
