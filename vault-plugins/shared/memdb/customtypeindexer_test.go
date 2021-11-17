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
	childTable    = "childTable"
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
									return nil, fmt.Errorf("need 'a', got %T", raw)
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
									return nil, fmt.Errorf("need 'c', got %T", raw)
								}
								return []byte(obj.UUID), nil
							},
						},
					},
				},
			},
			childTable: {
				Name: childTable,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "UUID",
						},
					},
				},
			},
		},
		CascadeDeletes: map[dataType][]Relation{
			childTable: {{
				OriginalDataTypeFieldName:     "UUID",
				RelatedDataType:               testTable,
				RelatedDataTypeFieldIndexName: childrenIndex,
				BuildRelatedCustomType: func(originalFieldValue interface{}) (customTypeValue interface{}, err error) {
					v, ok := originalFieldValue.(string)
					if !ok {
						return nil, fmt.Errorf("wrong type arg")
					}
					return c{
						UUID:    v,
						NotUUID: "",
					}, nil
				},
			}},
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

	obj, err := txn.First(testTable, PK, a{"u1", "u2"})

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

func getTxnWithData(t *testing.T) *Txn {
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
	return txn
}

func Test_GetCustomTypeSliceFieldIndexerTXN(t *testing.T) {
	txn := getTxnWithData(t)

	iter, err := txn.Get(testTable, childrenIndex, c{"u12", "nu12"})

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

func Test_DeleteCustomTypeSliceFieldIndexerTXNFail(t *testing.T) {
	txn := getTxnWithData(t)
	child := c{"u12", "nu12"}
	iter, err := txn.Get(testTable, childrenIndex, child)
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
	err = txn.Insert(childTable, &child)
	require.NoError(t, err)

	err = txn.Delete(childTable, &child)

	require.Error(t, err)
	require.Equal(t, err.Error(), "delete:not empty relation error:relation should be empty: {\"u12\" \"\"} found at table \"test_table\" by index \"children_index\"")
}
