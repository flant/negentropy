package memdb

import (
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/require"
)

func prepareTxn(t *testing.T) *Txn {
	schema := &DBSchema{
		Tables: testTables(),
		MandatoryForeignKeys: map[dataType][]Relation{
			childType1: {{
				"ParentUUID", parentType, PK,
			}},
			childType2: {{
				"ParentUUID", parentType, PK,
			}},
		},
		CascadeDeletes: map[dataType][]Relation{
			parentType: {{
				"UUID", childType2, parentTypeForeignKey,
			}},
		},
		CheckingRelations: map[dataType][]Relation{
			parentType: {{
				"UUID", childType1, parentTypeForeignKey,
			}},
		},
	}
	db, err := NewMemDB(schema)
	require.NoError(t, err)
	return db.Txn(true)
}

func prepareTxnWithParent(t *testing.T) (*Txn, *parent) {
	txn := prepareTxn(t)
	p := sampleParent()
	err := txn.Insert(parentType, p)
	require.NoError(t, err)
	return txn, p
}

func sampleParent() *parent {
	return &parent{
		UUID:       u1,
		Identifier: "parent1",
	}
}

func archivedParent() *parent {
	p := sampleParent()
	p.ArchivingTimestamp = 99
	p.ArchivingHash = 99
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
	c.ArchivingTimestamp = 99
	c.ArchivingHash = 99
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
	c.ArchivingTimestamp = 99
	c.ArchivingHash = 99
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

func Test_InsertOK(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
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
	txn, p := prepareTxnWithParent(t)

	err := txn.Delete(parentType, p)

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParent(), false)
}

func Test_DeleteFailCheckingRelations(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch1 := sampleChild1(p.UUID)
	err := txn.Insert(childType1, ch1)
	require.NoError(t, err)

	err = txn.Delete(parentType, sampleParent())

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParent(), true)
}

func Test_DeleteFailAtCascadeDeletes(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)

	err = txn.Delete(parentType, sampleParent())

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParent(), true)
}

func Test_CascadeDeleteOK(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)

	err = txn.CascadeDelete(parentType, sampleParent())

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParent(), false)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), false)
}

func Test_CascadeDeleteFailCheckingRelations(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	ch1 := sampleChild1(p.UUID)
	err = txn.Insert(childType1, ch1)
	require.NoError(t, err)

	err = txn.CascadeDelete(parentType, sampleParent())

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParent(), true)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), true)
	checkChild1Existence(t, txn, sampleChild1(p.UUID), true)
}

func Test_ArchiveOK(t *testing.T) {
	txn, _ := prepareTxnWithParent(t)

	err := txn.Archive(parentType, sampleParent(), 99, 99)

	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
}

func Test_ArchiveFailNotArchivableObject(t *testing.T) {
	txn := prepareTxn(t)
	x := 1

	err := txn.Archive(parentType, &x, 99, 99)

	require.ErrorIs(t, err, ErrNotArchivable)
}

func Test_ArchiveFailCheckingRelations(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch1 := sampleChild1(p.UUID)
	err := txn.Insert(childType1, ch1)
	require.NoError(t, err)

	err = txn.Archive(parentType, sampleParent(), 99, 99)

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParent(), true)
}

func Test_ArchiveFailAtCascadeDeletes(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)

	err = txn.Archive(parentType, sampleParent(), 99, 99)

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParent(), true)
}

func Test_CascadeArchiveOK(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)

	err = txn.CascadeArchive(parentType, sampleParent(), 99, 99)

	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)
}

func Test_CascadeArchiveFailCheckingRelations(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	ch1 := sampleChild1(p.UUID)
	err = txn.Insert(childType1, ch1)
	require.NoError(t, err)

	err = txn.CascadeArchive(parentType, sampleParent(), 99, 99)

	require.ErrorIs(t, err, ErrNotEmptyRelation)
	checkParentExistence(t, txn, sampleParent(), true)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), true)
	checkChild1Existence(t, txn, sampleChild1(p.UUID), true)
}

func Test_RestoreOK(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	err = txn.CascadeArchive(parentType, p, 99, 99)
	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)

	err = txn.Restore(parentType, sampleParent())

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)
}

func Test_RestoreFailChildWithoutParent(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	err = txn.CascadeArchive(parentType, p, 99, 99)
	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)

	err = txn.Restore(childType2, sampleChild2(p.UUID))

	require.ErrorIs(t, err, ErrForeignKey)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)
}

func Test_CascadeRestoreOK(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch2 := sampleChild2(p.UUID)
	err := txn.Insert(childType2, ch2)
	require.NoError(t, err)
	err = txn.CascadeArchive(parentType, p, 99, 99)
	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)

	err = txn.CascadeRestore(parentType, p)

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParent(), true)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), true)
}

func Test_CascadeRestoreOKJustWithRightTimeStampAndHash(t *testing.T) {
	txn, p := prepareTxnWithParent(t)
	ch22 := sampleChild2(p.UUID)
	ch22.UUID = u4
	err := txn.Insert(childType2, ch22)
	require.NoError(t, err)
	err = txn.Archive(childType2, ch22, 999, 999)
	require.NoError(t, err)
	archivedCh22 := sampleChild2(p.UUID)
	archivedCh22.UUID = u4
	archivedCh22.ArchivingTimestamp = 999
	archivedCh22.ArchivingHash = 999
	checkChild2Existence(t, txn, archivedCh22, true)

	ch2 := sampleChild2(p.UUID)
	err = txn.Insert(childType2, ch2)
	require.NoError(t, err)
	err = txn.CascadeArchive(parentType, p, 99, 99)
	require.NoError(t, err)
	checkParentExistence(t, txn, archivedParent(), true)
	checkChild2Existence(t, txn, archivedChild2(p.UUID), true)

	err = txn.CascadeRestore(parentType, p)

	require.NoError(t, err)
	checkParentExistence(t, txn, sampleParent(), true)
	checkChild2Existence(t, txn, sampleChild2(p.UUID), true)
	checkChild2Existence(t, txn, archivedCh22, true)
}

func Test_validateCyclicOK(t *testing.T) {
	rels := map[dataType]map[Relation]struct{}{
		"t1": {Relation{RelatedDataType: "t2"}: {}},
		"t2": {Relation{RelatedDataType: "t3"}: {}, Relation{RelatedDataType: "t4"}: {}},
		"t3": {Relation{RelatedDataType: "t4"}: {}},
		"t4": {Relation{RelatedDataType: "t5"}: {}},
	}

	err := validateCyclic("t1", rels)

	require.NoError(t, err)
}

func Test_validateCyclicFail(t *testing.T) {
	rels := map[dataType]map[Relation]struct{}{
		"t1": {Relation{RelatedDataType: "t2"}: {}},
		"t2": {Relation{RelatedDataType: "t3"}: {}, Relation{RelatedDataType: "t6"}: {}},
		"t3": {Relation{RelatedDataType: "t4"}: {}},
		"t4": {Relation{RelatedDataType: "t5"}: {}},
		"t5": {Relation{RelatedDataType: "t1"}: {}},
	}

	err := validateCyclic("t1", rels)

	require.Error(t, err)
	require.Equal(t, "dependencies chain:t1=>t2=>t3=>t4=>t5=>t1", err.Error())
}

func Test_validateForeignKeysFail(t *testing.T) {
	rels := map[dataType][]Relation{
		"t1": {{RelatedDataTypeFieldIndexName: "not_id"}},
	}

	err := validateForeignKeys(rels)

	require.Error(t, err)
	require.Equal(t, "invalid RelatedDataTypeFieldIndexName:not_id in FK:memdb.Relation{OriginalDataTypeFieldName:\"\", RelatedDataType:\"\", RelatedDataTypeFieldIndexName:\"not_id\"} of table t1", err.Error())
}

func Test_validateUniquenessChildRelationsFail(t *testing.T) {
	schema := &DBSchema{
		CascadeDeletes: map[dataType][]Relation{
			parentType: {{
				"UUID", childType2, parentTypeForeignKey,
			}},
		},
		CheckingRelations: map[dataType][]Relation{
			parentType: {{
				"UUID", childType2, parentTypeForeignKey,
			}},
		},
	}

	_, err := schema.validateUniquenessChildRelations()

	require.Error(t, err)
	require.Equal(t, "validateUniquenessChildRelations:relation memdb.Relation{OriginalDataTypeFieldName:\"UUID\", RelatedDataType:\"child2\", RelatedDataTypeFieldIndexName:\"parent_uuid\"} is repeated for parent dataType", err.Error())
}

func Test_validateExistenceIndexesFail(t *testing.T) {
	rels := map[dataType]map[Relation]struct{}{
		"t1": {Relation{
			OriginalDataTypeFieldName:     "ParentID",
			RelatedDataType:               parentType,
			RelatedDataTypeFieldIndexName: "no_index",
		}: {}},
	}

	err := validateExistenceIndexes(rels, testTables())

	require.Error(t, err)
	require.Equal(t, "index named 'no_index' not found at table 'parent', passed as relation to field 'ParentID' of table 't1'", err.Error())
}

func Test_validateCyclicFailForChildrenRels(t *testing.T) {
	schema := &DBSchema{
		Tables: testTables(),
		CascadeDeletes: map[dataType][]Relation{
			parentType: {{
				"UUID", childType2, parentTypeForeignKey,
			}},
		},
		CheckingRelations: map[dataType][]Relation{
			parentType: {{
				"UUID", childType2, parentTypeForeignKey,
			}},
		},
	}

	err := schema.Validate()

	require.ErrorIs(t, err, ErrInvalidSchema)
}

const (
	u1 = "00000000-0000-0000-0000-000000000001"
	u2 = "00000000-0000-0000-0000-000000000002"
	u3 = "00000000-0000-0000-0000-000000000003"
	u4 = "00000000-0000-0000-0000-000000000004"
)

const (
	parentType           = "parent"
	childType1           = "child1"
	childType2           = "child2"
	parentTypeForeignKey = "parent_uuid"
)

type parent struct {
	ArchivableImpl
	UUID       string `json:"uuid"` // PK
	Identifier string `json:"identifier"`
}

type child1 struct {
	ArchivableImpl
	UUID       string `json:"uuid"` // PK
	ParentUUID string `json:"parent_uuid"`
	Identifier string `json:"identifier"`
}

type child2 struct {
	ArchivableImpl
	UUID       string `json:"uuid"` // PK
	ParentUUID string `json:"parent_uuid"`
	Identifier string `json:"identifier"`
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
					Indexer: &memdb.UUIDFieldIndex{
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
					Indexer: &memdb.UUIDFieldIndex{
						Field: "ParentUUID",
					},
				},
			},
		},
	}
}
