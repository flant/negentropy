package memdb

import (
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/require"
)

const archivableType = "archivable"

type archivable struct {
	ArchiveMark
	UUID              string `json:"uuid"` // PK
	Identifier        string `json:"identifier"`
	UncontrolledField string
}

func (u *archivable) ObjType() string {
	return archivableType
}

func (u *archivable) ObjId() string {
	return u.UUID
}

const unarchivableType = "unarchivable"

type unarchivable struct {
	UUID              string `json:"uuid"` // PK
	Identifier        string `json:"identifier"`
	UncontrolledFiled string
}

const identifierIndex = "identifier_index"

func testShema() *DBSchema {
	return &DBSchema{
		Tables: map[string]*memdb.TableSchema{
			archivableType: {
				Name: archivableType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					identifierIndex: {
						Name: identifierIndex,
						Indexer: &memdb.StringFieldIndex{
							Field: "Identifier",
						},
					},
				},
			},
			unarchivableType: {
				Name: unarchivableType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					identifierIndex: {
						Name: identifierIndex,
						Indexer: &memdb.StringFieldIndex{
							Field: "Identifier",
						},
					},
				},
			},
		},
		UniqueConstraints: map[dataType][]indexName{
			"archivable":   {"identifier_index"},
			"unarchivable": {"identifier_index"},
		},
	}
}

const (
	uuid1       = "00000000-0000-0000-0000-000000000001"
	uuid2       = "00000000-0000-0000-0000-000000000002"
	identifier1 = "identifier1"
)

func filledDB(t *testing.T) *MemDB {
	db, err := NewMemDB(testShema())
	require.NoError(t, err)
	txn := db.Txn(true)
	err = txn.Insert(archivableType, &archivable{
		UUID:       uuid1,
		Identifier: identifier1,
	})
	require.NoError(t, err)
	err = txn.Insert(unarchivableType, &unarchivable{
		UUID:       uuid1,
		Identifier: identifier1,
	})
	require.NoError(t, err)
	txn.Commit()
	return db
}

func Test_archivableWrongWriting(t *testing.T) {
	txn := filledDB(t).Txn(true)

	err := txn.Insert(archivableType, &archivable{
		UUID:       uuid2,
		Identifier: identifier1, // repeat identifier
	})

	require.Error(t, err)
}

func Test_archivableValidWritingAfterDelete(t *testing.T) {
	db := filledDB(t)
	txn := db.Txn(true)
	err := txn.Insert(archivableType, &archivable{
		ArchiveMark: NewArchiveMark(), // archive
		UUID:        uuid1,
		Identifier:  identifier1,
	})
	require.NoError(t, err)

	err = txn.Insert(archivableType, &archivable{
		UUID:       uuid2,
		Identifier: identifier1, // repeat identifier for deleted record
	})

	require.NoError(t, err)
}

func Test_archivableUpdateWriting(t *testing.T) {
	txn := filledDB(t).Txn(true)

	err := txn.Insert(archivableType, &archivable{
		UUID:              uuid1,
		Identifier:        identifier1, // repeat identifier
		UncontrolledField: "test",
	})

	require.NoError(t, err)
}

func Test_unarchivableWrongWriting(t *testing.T) {
	txn := filledDB(t).Txn(true)

	err := txn.Insert(unarchivableType, &unarchivable{
		UUID:       uuid2,
		Identifier: identifier1, // repeat identifier
	})

	require.Error(t, err)
}

func Test_unarchivableValidWritingAfterDelete(t *testing.T) {
	db := filledDB(t)
	txn := db.Txn(true)
	ptr, err := txn.First(unarchivableType, PK, uuid1)
	require.NoError(t, err)
	err = txn.Delete(unarchivableType, ptr)
	require.NoError(t, err)

	err = txn.Insert(unarchivableType, &archivable{
		UUID:       uuid2,
		Identifier: identifier1, // repeat identifier for deleted record
	})

	require.NoError(t, err)
}
