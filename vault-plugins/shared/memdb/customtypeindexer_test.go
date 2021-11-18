package memdb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/require"
)

type a struct {
	UUID    string
	NotUUID string
}

type b struct {
	ID       a
	Name     string
	Children []c
}

type c struct {
	UUID    string
	NotUUID string
}

const (
	testTable     = "test_table"
	childrenIndex = "children_index"
)

func getTXN(t *testing.T) *Txn {
	schema := &DBSchema{
		Tables: map[string]*memdb.TableSchema{
			testTable: {
				Name: testTable,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &CustomTypeFieldIndexer{
							Field: "ID",
							FromCustomType: func(raw interface{}) ([]byte, error) {
								obj, ok := raw.(a)
								if !ok {
									obj, ok := raw.(*a)
									if !ok {
										return nil, fmt.Errorf("wrong type")
									}
									return []byte(obj.UUID), nil
								}
								return []byte(obj.UUID), nil
							},
						},
					},
					childrenIndex: {
						Name:         childrenIndex,
						AllowMissing: true,
						Unique:       false,
						Indexer: &CustomTypeSliceFieldIndexer{
							Field: "Children",
							FromCustomType: func(raw interface{}) ([]byte, error) {
								obj, ok := raw.(c)
								if !ok {
									obj, ok := raw.(*c)
									if !ok {
										return nil, fmt.Errorf("wrong type")
									}
									return []byte(obj.UUID), nil
								}
								return []byte(obj.UUID), nil
							},
						},
					},
				},
			},
		},
	}
	db, err := NewMemDB(schema)
	require.NoError(t, err)
	txn := db.Txn(true)
	return txn
}

func Test_InsertCustomTypeFieldIndexer(t *testing.T) {
	txn := getTXN(t)

	err := txn.Insert(testTable, &b{
		ID:   a{"u1", "u2"},
		Name: "u3",
	})

	require.NoError(t, err)
}

func Test_FirstCustomTypeFieldIndexer(t *testing.T) {
	txn := getTXN(t)
	err := txn.Insert(testTable, &b{
		ID:   a{"u1", "u2"},
		Name: "u3",
	})
	require.NoError(t, err)

	obj, err := txn.First(testTable, PK, &a{"u1", "u2"})

	require.NoError(t, err)
	require.Equal(t, &b{
		ID:   a{"u1", "u2"},
		Name: "u3",
	}, obj)
}

func Test_InsertCustomTypeSliceFieldIndexerTXN(t *testing.T) {
	txn := getTXN(t)

	err := txn.Insert(testTable, &b{
		ID:       a{"u1", "u2"},
		Name:     "u3",
		Children: []c{{"u11", "nu11"}, {"u12", "nu12"}},
	})

	require.NoError(t, err)
}

func Test_GetCustomTypeSliceFieldIndexerTXN(t *testing.T) {
	txn := getTXN(t)
	err := txn.Insert(testTable, &b{
		ID:       a{"u1", "nu1"},
		Name:     "n1",
		Children: []c{{"u11", "nu11"}, {"u12", "nu12"}},
	})
	require.NoError(t, err)
	err = txn.Insert(testTable, &b{
		ID:       a{"u2", "nu2"},
		Name:     "n2",
		Children: []c{{"u11", "nu11"}, {"u13", "nu13"}},
	})
	require.NoError(t, err)
	err = txn.Insert(testTable, &b{
		ID:       a{"u3", "u3"},
		Name:     "n3",
		Children: []c{{"u12", "nu12"}, {"u13", "nu13"}},
	})
	require.NoError(t, err)

	iter, err := txn.Get(testTable, childrenIndex, &c{"u12", "nu12"})

	require.NoError(t, err)

	r1 := iter.Next()
	require.NotEmpty(t, r1)
	b1 := r1.(*b)
	require.Equal(t, "n1", b1.Name)
	r2 := iter.Next()
	require.NotEmpty(t, r2)
	b2 := r2.(*b)
	require.Equal(t, "n3", b2.Name)
	r3 := iter.Next()
	require.Empty(t, r3)
}
