package memdb

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/require"
)

func prepareTxn(t *testing.T) *Txn {
	schema := &DBSchema{
		Tables: testTables(),
		MandatoryForeignKeys: map[DataType][]Relation{
			childType1: {{
				OriginalDataTypeFieldName: "ParentUUID", RelatedDataType: parentType, RelatedDataTypeFieldIndexName: PK,
			}},
			childType2: {{
				OriginalDataTypeFieldName: "ParentUUID", RelatedDataType: parentType, RelatedDataTypeFieldIndexName: PK,
			}},
			childType3: {{
				OriginalDataTypeFieldName: "Parents", RelatedDataType: parentType, RelatedDataTypeFieldIndexName: PK,
			}},
		},
		CascadeDeletes: map[DataType][]Relation{
			parentType: {
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: childType2, RelatedDataTypeFieldIndexName: parentTypeForeignKey},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: childType3, RelatedDataTypeFieldIndexName: parentsIndex},
			},
		},
		CheckingRelations: map[DataType][]Relation{
			parentType: {{
				OriginalDataTypeFieldName: "UUID", RelatedDataType: childType1, RelatedDataTypeFieldIndexName: parentTypeForeignKey,
			}},
		},
	}
	db, err := NewMemDB(schema)
	require.NoError(t, err)
	return db.Txn(true)
}

func prepareTxnWithParentU1(t *testing.T) (*Txn, *parent) {
	txn := prepareTxn(t)
	p := sampleParentU1()
	err := txn.Insert(parentType, p)
	require.NoError(t, err)
	return txn, p
}

func sampleParentU1() *parent {
	return &parent{
		UUID:       u1,
		Identifier: "parent1",
	}
}

func archivedParent() *parent {
	p := sampleParentU1()
	p.Timestamp = 99
	p.Hash = 99
	return p
}

func sampleChild1(parentUUID string) *child1 {
	return &child1{
		UUID:       u2,
		ParentUUID: parentUUID,
		Identifier: "child1",
	}
}

func archivedChild1(parentUUID string) *child1 {
	c := sampleChild1(parentUUID)
	c.Timestamp = 99
	c.Hash = 99
	return c
}

func sampleChild2(parentUUID string) *child2 {
	return &child2{
		UUID:       u3,
		ParentUUID: parentUUID,
		Identifier: "child2",
	}
}

func archivedChild2(parentUUID string) *child2 {
	c := sampleChild2(parentUUID)
	c.Timestamp = 99
	c.Hash = 99
	return c
}

func checkParentExistence(t *testing.T, txn *Txn, parent *parent, shouldExists bool) {
	expectedP, err := txn.First(parentType, PK, parent.UUID)
	require.NoError(t, err)
	if shouldExists {
		require.NotEmpty(t, expectedP)
		require.Equal(t, parent, expectedP)
	} else {
		require.Empty(t, expectedP)
	}
}

func checkChild1Existence(t *testing.T, txn *Txn, child1 *child1, shouldExists bool) {
	expectedChild, err := txn.First(childType1, PK, child1.UUID)
	require.NoError(t, err)
	if shouldExists {
		require.NotEmpty(t, expectedChild)
		require.Equal(t, child1, expectedChild)
	} else {
		require.Empty(t, expectedChild)
	}
}

func checkChild2Existence(t *testing.T, txn *Txn, child2 *child2, shouldExists bool) {
	expectedChild, err := txn.First(childType2, PK, child2.UUID)
	require.NoError(t, err)
	if shouldExists {
		require.NotEmpty(t, expectedChild)
		require.Equal(t, child2, expectedChild)
	} else {
		require.Empty(t, expectedChild)
	}
}

var standardArchiveMark = ArchiveMark{
	Timestamp: 99,
	Hash:      99,
}

func Test_InsertOK(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch := sampleChild1(p.UUID)

	err := txn.Insert(childType1, ch)

	require.NoError(t, err)
	checkChild1Existence(t, txn, sampleChild1(p.UUID), true)
}

func Test_InsertFailForeignKey(t *testing.T) {
	txn := prepareTxn(t)
	ch := sampleChild1(u1)

	err := txn.Insert(childType1, ch)

	require.ErrorIs(t, err, ErrForeignKey)
	checkChild1Existence(t, txn, ch, false)
}

func Test_DeleteOK(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)

	err := txn.Delete(parentType, p)

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParentU1(), false)
}

func Test_DeleteFailCheckingRelations(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch1 := sampleChild1(p.UUID)
	err := txn.Insert(childType1, ch1)
	require.NoError(t, err)

	err = txn.Delete(parentType, sampleParentU1())

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParentU1(), true)
}

func Test_DeleteFailAtCascadeDeletes(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)

	err = txn.Delete(parentType, sampleParentU1())

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParentU1(), true)
}

func Test_CascadeDeleteOK(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)

	err = txn.CascadeDelete(parentType, sampleParentU1())

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParentU1(), false)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), false)
}

func Test_CascadeDeleteFailCheckingRelations(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	ch1 := sampleChild1(p.UUID)
	err = txn.Insert(childType1, ch1)
	require.NoError(t, err)

	err = txn.CascadeDelete(parentType, sampleParentU1())

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParentU1(), true)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), true)
	checkChild1Existence(t, txn, sampleChild1(p.UUID), true)
}

func Test_ArchiveOK(t *testing.T) {
	txn, _ := prepareTxnWithParentU1(t)

	err := txn.Archive(parentType, sampleParentU1(), standardArchiveMark)

	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
}

func Test_ArchiveFailNotArchivableObject(t *testing.T) {
	txn := prepareTxn(t)
	x := 1

	err := txn.Archive(parentType, &x, standardArchiveMark)

	require.ErrorIs(t, err, ErrNotArchivable)
}

func Test_ArchiveFailCheckingRelations(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch1 := sampleChild1(p.UUID)
	err := txn.Insert(childType1, ch1)
	require.NoError(t, err)

	err = txn.Archive(parentType, sampleParentU1(), standardArchiveMark)

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParentU1(), true)
}

func Test_ArchiveFailAtCascadeDeletes(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)

	err = txn.Archive(parentType, sampleParentU1(), standardArchiveMark)

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParentU1(), true)
}

func Test_CascadeArchiveOK(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)

	err = txn.CascadeArchive(parentType, sampleParentU1(), standardArchiveMark)

	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)
}

func Test_CascadeArchiveFailCheckingRelations(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	ch1 := sampleChild1(p.UUID)
	err = txn.Insert(childType1, ch1)
	require.NoError(t, err)

	err = txn.CascadeArchive(parentType, sampleParentU1(), standardArchiveMark)

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParentU1(), true)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), true)
	checkChild1Existence(t, txn, sampleChild1(p.UUID), true)
}

func Test_RestoreOK(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	err = txn.CascadeArchive(parentType, p, standardArchiveMark)
	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)

	err = txn.Restore(parentType, sampleParentU1())

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParentU1(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)
}

func Test_RestoreFailChildWithoutParent(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	err = txn.CascadeArchive(parentType, p, standardArchiveMark)
	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)

	err = txn.Restore(childType2, sampleChild2(p.UUID))

	require.ErrorIs(t, err, ErrForeignKey)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)
}

func Test_CascadeRestoreOK(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	err = txn.CascadeArchive(parentType, p, standardArchiveMark)
	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)

	err = txn.CascadeRestore(parentType, p)

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParentU1(), true)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), true)
}

func Test_CascadeRestoreOKJustWithRightTimeStampAndHash(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	ch22 := sampleChild2(p.UUID)
	ch22.UUID = u4
	err := txn.Insert(childType2, ch22)
	require.NoError(t, err)
	err = txn.Archive(childType2, ch22, ArchiveMark{999, 999})
	require.NoError(t, err)
	archivedCh22 := sampleChild2(p.UUID)
	archivedCh22.UUID = u4
	archivedCh22.Timestamp = 999
	archivedCh22.Hash = 999
	checkChild2Existence(t, txn, archivedCh22, true)

	ch2 := sampleChild2(p.UUID)
	err = txn.Insert(childType2, ch2)
	require.NoError(t, err)
	err = txn.CascadeArchive(parentType, p, standardArchiveMark)
	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)

	err = txn.CascadeRestore(parentType, p)

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParentU1(), true)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), true)
	checkChild2Existence(t, txn, archivedCh22, true)
}

func Test_validateCyclicOK(t *testing.T) {
	rels := map[DataType]map[RelationKey]struct{}{
		"t1": {RelationKey{RelatedDataType: "t2"}: {}},
		"t2": {RelationKey{RelatedDataType: "t3"}: {}, RelationKey{RelatedDataType: "t4"}: {}},
		"t3": {RelationKey{RelatedDataType: "t4"}: {}},
		"t4": {RelationKey{RelatedDataType: "t5"}: {}},
	}

	err := validateCyclic("t1", rels)

	require.NoError(t, err)
}

func Test_validateCyclicFail(t *testing.T) {
	rels := map[DataType]map[RelationKey]struct{}{
		"t1": {RelationKey{RelatedDataType: "t2"}: {}},
		"t2": {RelationKey{RelatedDataType: "t3"}: {}, RelationKey{RelatedDataType: "t6"}: {}},
		"t3": {RelationKey{RelatedDataType: "t4"}: {}},
		"t4": {RelationKey{RelatedDataType: "t5"}: {}},
		"t5": {RelationKey{RelatedDataType: "t1"}: {}},
	}

	err := validateCyclic("t1", rels)

	require.Error(t, err)
	require.Equal(t, "dependencies chain:t1=>t2=>t3=>t4=>t5=>t1", err.Error())
}

func Test_validateForeignKeysFail(t *testing.T) {
	rels := map[DataType][]Relation{
		"t1": {{RelatedDataTypeFieldIndexName: "not_id"}},
	}

	err := validateForeignKeys(rels)

	require.Error(t, err)
	require.Equal(t, "invalid RelatedDataTypeFieldIndexName:not_id in FK:memdb.Relation{OriginalDataTypeFieldName:\"\", RelatedDataType:\"\", RelatedDataTypeFieldIndexName:\"not_id\", indexIsSliceFieldIndex:false, BuildRelatedCustomType:(func(interface {}) (interface {}, error))(nil)} of table t1", err.Error())
}

func Test_validateUniquenessChildRelationsFail(t *testing.T) {
	schema := &DBSchema{
		CascadeDeletes: map[DataType][]Relation{
			parentType: {{
				OriginalDataTypeFieldName: "UUID", RelatedDataType: childType2, RelatedDataTypeFieldIndexName: parentTypeForeignKey,
			}},
		},
		CheckingRelations: map[DataType][]Relation{
			parentType: {{
				OriginalDataTypeFieldName: "UUID", RelatedDataType: childType2, RelatedDataTypeFieldIndexName: parentTypeForeignKey,
			}},
		},
	}

	_, err := schema.validateUniquenessChildRelations()

	require.Error(t, err)
	require.Equal(t, "validateUniquenessChildRelations:relation memdb.Relation{OriginalDataTypeFieldName:\"UUID\", RelatedDataType:\"child2\", RelatedDataTypeFieldIndexName:\"parent_uuid\", indexIsSliceFieldIndex:false, BuildRelatedCustomType:(func(interface {}) (interface {}, error))(nil)} is repeated for parent DataType", err.Error())
}

func Test_validateExistenceIndexesFail(t *testing.T) {
	rels := map[DataType][]Relation{
		"t1": {Relation{
			OriginalDataTypeFieldName:     "ParentID",
			RelatedDataType:               parentType,
			RelatedDataTypeFieldIndexName: "no_index",
		}},
	}

	err := (&DBSchema{Tables: testTables(), MandatoryForeignKeys: rels}).validateExistenceIndexes()

	require.Error(t, err)
	require.Equal(t, "index named \"no_index\" not found at table \"parent\", passed as relation to field \"ParentID\" of table \"t1\"", err.Error())
}

func Test_validateCyclicFailForChildrenRels(t *testing.T) {
	schema := &DBSchema{
		Tables: testTables(),
		CascadeDeletes: map[DataType][]Relation{
			parentType: {{
				OriginalDataTypeFieldName: "UUID", RelatedDataType: childType2, RelatedDataTypeFieldIndexName: parentTypeForeignKey,
			}},
		},
		CheckingRelations: map[DataType][]Relation{
			parentType: {{
				OriginalDataTypeFieldName: "UUID", RelatedDataType: childType2, RelatedDataTypeFieldIndexName: parentTypeForeignKey,
			}},
		},
	}

	err := schema.Validate()

	require.ErrorIs(t, err, ErrInvalidSchema)
}

func Test_CleanChildrenSliceIndexes(t *testing.T) {
	txn, p := prepareTxnWithParentU1(t)
	for i, uuid := range []string{u2, u3} {
		err := txn.Insert(parentType, &parent{
			UUID:       uuid,
			Identifier: fmt.Sprintf("parent_%d", i),
		})
		require.NoError(t, err)
	}
	err := txn.Insert(childType3, &child3{
		UUID:    u4,
		Parents: []string{u2, u1},
	})
	require.NoError(t, err)
	err = txn.Insert(childType3, &child3{
		UUID:    u5,
		Parents: []string{u2, u1, u3},
	})
	require.NoError(t, err)

	err = txn.CleanChildrenSliceIndexes(parentType, p)
	// Deleting after cleaning
	err = txn.CascadeArchive(parentType, p, NewArchiveMark())
	require.NoError(t, err)

	require.NoError(t, err)
	obj, err := txn.First(childType3, PK, u4)
	require.NoError(t, err)
	ch3, ok := obj.(*child3)
	require.True(t, ok)
	require.Equal(t, []string{u2}, ch3.Parents)
	require.Equal(t, ch3.Hash, int64(0))

	obj, err = txn.First(childType3, PK, u5)
	require.NoError(t, err)
	ch3, ok = obj.(*child3)
	require.True(t, ok)
	require.Equal(t, []string{u2, u3}, ch3.Parents)
	require.Equal(t, ch3.Hash, int64(0))
}

func Test_checkPtrAndReturnIndirect(t *testing.T) {
	var a int = 100
	var objPtr interface{} = &a
	var intObj interface{} = a

	obj, err := checkPtrAndReturnIndirect(objPtr)

	require.NoError(t, err)
	require.IsType(t, intObj, obj)
}

func Test_checkPtrAndReturnIndirectFail(t *testing.T) {
	var a int = 100
	var notObjPtr interface{} = a

	obj, err := checkPtrAndReturnIndirect(notObjPtr)

	require.ErrorIs(t, err, ErrNotPtr)
	require.Empty(t, obj)
}

const (
	u1 = "00000000-0000-4000-A000-000000000001"
	u2 = "00000000-0000-4000-A000-000000000002"
	u3 = "00000000-0000-4000-A000-000000000003"
	u4 = "00000000-0000-4000-A000-000000000004"
	u5 = "00000000-0000-4000-A000-000000000005"
)

const (
	parentType           = "parent"
	childType1           = "child1"
	childType2           = "child2"
	childType3           = "child3"
	parentTypeForeignKey = "parent_uuid"
	parentsIndex         = "parents_index"
)

type parent struct {
	ArchiveMark
	UUID       string `json:"uuid"` // PK
	Identifier string `json:"identifier"`
}

type child1 struct {
	ArchiveMark
	UUID       string `json:"uuid"` // PK
	ParentUUID string `json:"parent_uuid"`
	Identifier string `json:"identifier"`
}

type child2 struct {
	ArchiveMark
	UUID       string `json:"uuid"` // PK
	ParentUUID string `json:"parent_uuid"`
	Identifier string `json:"identifier"`
}

type child3 struct {
	ArchiveMark
	UUID    string   `json:"uuid"` // PK
	Parents []string `json:"parents"`
}

func testTables() map[string]*memdb.TableSchema {
	return map[string]*memdb.TableSchema{
		parentType: {
			Name: parentType,
			Indexes: map[string]*memdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &memdb.UUIDFieldIndex{
						Field: "UUID",
					},
				},
			},
		},
		childType1: {
			Name: childType1,
			Indexes: map[string]*memdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &memdb.UUIDFieldIndex{
						Field: "UUID",
					},
				},
				parentTypeForeignKey: {
					Name: parentTypeForeignKey,
					Indexer: &memdb.StringFieldIndex{
						Field: "ParentUUID",
					},
				},
			},
		},
		childType2: {
			Name: childType2,
			Indexes: map[string]*memdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &memdb.UUIDFieldIndex{
						Field: "UUID",
					},
				},
				parentTypeForeignKey: {
					Name: parentTypeForeignKey,
					Indexer: &memdb.StringFieldIndex{
						Field: "ParentUUID",
					},
				},
			},
		},
		childType3: {
			Name: childType3,
			Indexes: map[string]*memdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &memdb.UUIDFieldIndex{
						Field: "UUID",
					},
				},
				parentsIndex: {
					Name: parentsIndex,
					Indexer: &memdb.StringSliceFieldIndex{
						Field: "Parents",
					},
				},
			},
		},
	}
}

func Test_FirstUnExisted(t *testing.T) {
	txn, _ := prepareTxnWithParentU1(t)

	p, err := txn.First(parentType, "id", u2)

	require.NoError(t, err)
	require.Nil(t, p)
}

func Test_DeleteUnExisted(t *testing.T) {
	txn, _ := prepareTxnWithParentU1(t)
	unexisted := &parent{
		UUID:       u2,
		Identifier: "parent2",
	}

	err := txn.Delete(parentType, unexisted)

	require.ErrorIs(t, err, memdb.ErrNotFound)
}
