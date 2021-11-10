package memdb

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/go-memdb"
)

const PK = "id"

type (
	dataType  = string
	fieldName = string
	indexName = string
	UnixTime  = int64
)

var (
	ErrForeignKey       = fmt.Errorf("foreign key error")
	ErrNotEmptyRelation = fmt.Errorf("not empty relation error")
	ErrNotArchivable    = fmt.Errorf("not archivable object")
	ErrInvalidSchema    = fmt.Errorf("invalid DBSchema")
)

type Relation struct {
	OriginalDataTypeFieldName     fieldName
	RelatedDataType               dataType
	RelatedDataTypeFieldIndexName indexName
}

type Archivable interface {
	Archive(timeStamp UnixTime, archivingHash int64)
	Restore()
	Archived() bool
	ArchiveMarks() (timeStamp UnixTime, archivingHash int64)
}

type ArchivableImpl struct {
	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (a *ArchivableImpl) Archive(timeStamp UnixTime, hash int64) {
	a.ArchivingTimestamp = timeStamp
	a.ArchivingHash = hash
}

func (a *ArchivableImpl) Restore() {
	a.ArchivingTimestamp = 0
	a.ArchivingHash = 0
}

func (a *ArchivableImpl) Archived() bool {
	return a.ArchivingTimestamp != 0
}

func (a *ArchivableImpl) ArchiveMarks() (timeStamp UnixTime, archivingHash int64) {
	return a.ArchivingTimestamp, a.ArchivingHash
}

type DBSchema struct {
	*memdb.DBSchema
	// check at Insert
	// prohibited to use the same dataType as map key and as value in Relation.RelatedDataType
	MandatoryForeignKeys map[dataType][]Relation
	// use at CascadeDelete
	// check at Delete, deleting fails if any of relation is not empty
	// prohibited to use the same dataType as map key and as value in Relation.RelatedDataType
	CascadeDeletes map[dataType][]Relation
	// check at CascadeDelete and Delete, deleting fails if any of relation is not empty
	// prohibited to place the same Relation into CascadeDeletes and CheckingRelations
	// prohibited to use the same dataType as map key and as value in Relation.RelatedDataType
	CheckingRelations map[dataType][]Relation
}

func (s *DBSchema) validate() error {
	if err := s.DBSchema.Validate(); err != nil {
		return fmt.Errorf("%w:%s", ErrInvalidSchema, err)
	}
	if err := validateForeignKeys(s.MandatoryForeignKeys); err != nil {
		return fmt.Errorf("%w:%s", ErrInvalidSchema, err)
	}
	allChildRelations, err := s.validateUniquenessChildRelations()
	if err != nil {
		return fmt.Errorf("%w:%s", ErrInvalidSchema, err)
	}
	err = validateExistenceIndexes(allChildRelations, s.DBSchema)
	if err != nil {
		return fmt.Errorf("%w:%s", ErrInvalidSchema, err)
	}
	for dt := range allChildRelations {
		err = validateCyclic(dt, allChildRelations)
		if err != nil {
			return fmt.Errorf("%w:%s", ErrInvalidSchema, err)
		}
	}
	return nil
}

// validateForeignKeys checks:
// 1) absence the same dataType as key and RelatedDataType
// 2) absence of cyclic dependencies
// 3) only 'id' as RelatedDataTypeFieldIndexName
func validateForeignKeys(fks map[dataType][]Relation) error {
	type childDataType = dataType
	rels := map[dataType]map[Relation]struct{}{}
	for d, keys := range fks {
		ks, ok := rels[d]
		if !ok {
			ks = map[Relation]struct{}{}
		}
		for _, key := range keys {
			if key.RelatedDataTypeFieldIndexName != PK {
				return fmt.Errorf("invalid RelatedDataTypeFieldIndexName:%s in FK:%#v of table %s",
					key.RelatedDataTypeFieldIndexName, key, d)
			}
			ks[key] = struct{}{}
		}
		rels[d] = ks
	}
	for d := range rels {
		err := validateCyclic(d, rels)
		if err != nil {
			return fmt.Errorf("cyclic rependency: %s",
				err.Error())
		}
	}
	return nil
}

// validateCyclic checks absence of cyclic dependencies between tables
func validateCyclic(topDataType dataType, rels map[dataType]map[Relation]struct{}) error {
	type dependency struct {
		parentType     dataType
		curIdx         int
		childDataTypes []dataType
	}
	childDataTypesFunc := func(parentDataType dataType, rels map[dataType]map[Relation]struct{}) []dataType {
		mapResult := map[dataType]struct{}{}
		for r := range rels[parentDataType] {
			mapResult[r.RelatedDataType] = struct{}{}
		}
		result := make([]dataType, 0, len(mapResult))
		for r := range mapResult {
			result = append(result, r)
		}
		return result
	}
	deps := make([]dependency, len(rels)+1)
	deps[0] = dependency{
		parentType:     topDataType,
		curIdx:         0,
		childDataTypes: childDataTypesFunc(topDataType, rels),
	}
	l := 0
loop:
	for deps[l].curIdx < len(deps[l].childDataTypes) || l != 0 {
		curIdx := deps[l].curIdx
		switch {
		case curIdx >= len(deps[l].childDataTypes):
			l--
			deps[l].curIdx++
			continue loop
		case deps[l].childDataTypes[curIdx] == topDataType:
			// create chain
			chainBuilder := strings.Builder{}
			for i := 0; i <= l; i++ {
				if chainBuilder.Len() != 0 {
					chainBuilder.WriteString("=>")
				}
				chainBuilder.WriteString(deps[i].parentType)
			}
			chainBuilder.WriteString("=>" + deps[0].parentType)
			return fmt.Errorf("dependencies chain:%s", chainBuilder.String())
		case len(rels[deps[l].childDataTypes[curIdx]]) > 0:
			curType := deps[l].childDataTypes[curIdx]
			l++
			deps[l] = dependency{
				parentType:     curType,
				curIdx:         0,
				childDataTypes: childDataTypesFunc(curType, rels),
			}
		default:
			deps[l].curIdx++
		}
	}
	return nil
}

func validateExistenceIndexes(rels map[dataType]map[Relation]struct{}, schema *memdb.DBSchema) error {
	for dt, rsSet := range rels {
		for r := range rsSet {
			if _, ok := schema.Tables[r.RelatedDataType]; !ok {
				return fmt.Errorf("table %s is absent in memdb.DBSchema", dt)
			} else {
				if _, ok := schema.Tables[r.RelatedDataType].Indexes[r.RelatedDataTypeFieldIndexName]; ok {
					break
				}
				return fmt.Errorf("index named '%s' not found at table '%s', passed as relation to field '%s' of table '%s'",
					r.RelatedDataTypeFieldIndexName, r.RelatedDataType, r.OriginalDataTypeFieldName, dt)
			}
		}
	}
	return nil
}

// validateUniquenessChildRelations checks uniqueness for CascadeDeletes and CheckingRelations
// returns united map of relations
func (s *DBSchema) validateUniquenessChildRelations() (map[dataType]map[Relation]struct{}, error) {
	allRels := map[dataType]map[Relation]struct{}{}
	allRels, err := checkRsMapForRepeating(allRels, s.CascadeDeletes)
	if err != nil {
		return nil, fmt.Errorf("validateUniquenessChildRelations:%w", err)
	}
	allRels, err = checkRsMapForRepeating(allRels, s.CheckingRelations)
	if err != nil {
		return nil, fmt.Errorf("validateUniquenessChildRelations:%w", err)
	}
	return allRels, nil
}

// checks ForRepeating checks repeating of relations
// returns map of relation
func checkRsMapForRepeating(allRels map[dataType]map[Relation]struct{},
	rsMap map[dataType][]Relation) (map[dataType]map[Relation]struct{}, error) {
	for d, rs := range rsMap {
		if rels, ok := allRels[d]; ok {
			for _, r := range rs {
				if _, rep := rels[r]; rep {
					return nil, fmt.Errorf("relation %#v is repeated for %s dataType", r, d)
				}
				rels[r] = struct{}{}
			}
		} else {
			rels := map[Relation]struct{}{}
			for _, r := range rs {
				rels[r] = struct{}{}
			}
			allRels[d] = rels
		}
	}
	return allRels, nil
}

type MemDB struct {
	*memdb.MemDB

	schema *DBSchema
}

type Txn struct {
	*memdb.Txn

	schema *DBSchema
}

func (m *MemDB) Txn(write bool) *Txn {
	mTxn := m.MemDB.Txn(write)
	if write {
		mTxn.TrackChanges()
	}
	return &Txn{Txn: mTxn, schema: m.schema}
}

func (t *Txn) Insert(table string, obj interface{}) error {
	err := t.checkForeignKeys(table, obj)
	if err != nil {
		return fmt.Errorf("insert:%w", err)
	}
	return t.Txn.Insert(table, obj)
}

func (t *Txn) Delete(table string, obj interface{}) error {
	err := t.checkCascadeDeletesAndCheckingRelations(table, obj)
	if err != nil {
		return fmt.Errorf("delete:%w", err)
	}
	err = t.Txn.Delete(table, obj)
	if err != nil {
		return fmt.Errorf("delete:%w", err)
	}
	return nil
}

func (t *Txn) CascadeDelete(table string, obj interface{}) error {
	err := t.checkCheckingRelations(table, obj)
	if err != nil {
		return fmt.Errorf("cascadeDelete:%w", err)
	}
	err = t.processRelations(t.schema.CascadeDeletes[table], obj, t.deleteChild, ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cascadeDelete:%w", err)
	}
	err = t.Txn.Delete(table, obj)
	if err != nil {
		return fmt.Errorf("cascadeDelete:%w", err)
	}
	return nil
}

func (t *Txn) Archive(table string, obj interface{}, archivingTimeStamp int64, archivingHash int64) error {
	a, ok := obj.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, obj)
	}
	err := t.checkCascadeDeletesAndCheckingRelations(table, obj)
	if err != nil {
		return fmt.Errorf("archive:%w", err)
	}
	a.Archive(archivingTimeStamp, archivingHash)
	err = t.Insert(table, obj)
	if err != nil {
		return fmt.Errorf("archive:%w", err)
	}
	return nil
}

func (t *Txn) CascadeArchive(table string, obj interface{}, archivingTimeStamp int64, archivingHash int64) error {
	a, ok := obj.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, obj)
	}
	err := t.checkCheckingRelations(table, obj)
	if err != nil {
		return fmt.Errorf("cascadeArchive:%w", err)
	}
	err = t.processRelations(t.schema.CascadeDeletes[table], obj, t.archiveChild(archivingTimeStamp, archivingHash), ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cascadeArchive:%w", err)
	}
	a.Archive(archivingTimeStamp, archivingHash)
	err = t.Insert(table, obj)
	if err != nil {
		return fmt.Errorf("cascadeArchive:%w", err)
	}
	return nil
}

func (t *Txn) Restore(table string, obj interface{}) error {
	a, ok := obj.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, obj)
	}
	a.Restore()
	err := t.Insert(table, obj)
	if err != nil {
		return fmt.Errorf("restore:%w", err)
	}
	return nil
}

func (t *Txn) CascadeRestore(table string, obj interface{}) error {
	a, ok := obj.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, obj)
	}
	timeStamp, archivingHash := a.ArchiveMarks()
	a.Restore()
	err := t.Insert(table, obj)
	if err != nil {
		return fmt.Errorf("cascadeRestore:%w", err)
	}
	err = t.processRelations(t.schema.CascadeDeletes[table], obj, t.restoreChild(timeStamp, archivingHash), ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cascadeRestore:%w", err)
	}
	return nil
}

func (t *Txn) checkForeignKeys(table string, obj interface{}) error {
	keys := t.schema.MandatoryForeignKeys[table]
	return t.processRelations(keys, obj, t.checkForeignKey, ErrForeignKey)
}

func (t *Txn) checkForeignKey(checkedFieldValue interface{}, key Relation) error {
	relatedRecord, err := t.First(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, checkedFieldValue)
	if err != nil {
		return fmt.Errorf("getting related record:%w", err)
	}
	a, ok := relatedRecord.(Archivable)
	if !ok {
		return fmt.Errorf("%w related record %#v is not archivable", ErrNotArchivable, relatedRecord)
	}
	if relatedRecord == nil || a.Archived() {
		return fmt.Errorf("FK violation: '%s' not found at table '%s' at index '%s'",
			checkedFieldValue, key.RelatedDataType, key.RelatedDataTypeFieldIndexName)
	}
	return nil
}

func (t *Txn) checkCascadeDeletesAndCheckingRelations(table string, obj interface{}) error {
	rels := append(t.schema.CascadeDeletes[table],
		t.schema.CheckingRelations[table]...)
	return t.processRelations(rels, obj, t.checkRelationShouldBeEmpty, ErrNotEmptyRelation)
}

func (t *Txn) checkCheckingRelations(table string, obj interface{}) error {
	rels := t.schema.CheckingRelations[table]
	return t.processRelations(rels, obj, t.checkRelationShouldBeEmpty, ErrNotEmptyRelation)
}

// implement main loop checking relations
func (t *Txn) processRelations(relations []Relation, obj interface{},
	relationHandler func(originObjectFieldValue interface{}, key Relation) error, relationHandlerError error) error {
	valueIface := reflect.ValueOf(obj)
	if valueIface.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("obj `%s` is not ptr", valueIface.Type())
	}
	msgs := []string{}
	for _, key := range relations {
		field := valueIface.Elem().FieldByName(key.OriginalDataTypeFieldName)
		if !field.IsValid() {
			return fmt.Errorf("obj `%s` does not have the field `%s`", valueIface.Type(), key.OriginalDataTypeFieldName)
		}
		checkedFieldValue := field.Interface()
		if err := relationHandler(checkedFieldValue, key); err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	builder := strings.Builder{}
	if len(msgs) != 0 {
		for _, msg := range msgs {
			if builder.Len() != 0 {
				builder.WriteString(";")
			}
			builder.WriteString(msg)
		}
		return fmt.Errorf(builder.String()+":%w", relationHandlerError)
	}
	return nil
}

func (t *Txn) checkRelationShouldBeEmpty(checkedFieldValue interface{}, key Relation) error {
	relatedRecord, err := t.First(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, checkedFieldValue)
	if err != nil {
		return fmt.Errorf("getting related record:%w", err)
	}
	if relatedRecord != nil {
		return fmt.Errorf("relation should be empty: '%s' found at table '%s' by index '%s'",
			checkedFieldValue, key.RelatedDataType, key.RelatedDataTypeFieldIndexName)
	}
	return nil
}

func (t *Txn) deleteChild(parentObjectFiledValue interface{}, key Relation) error {
	relatedRecord, err := t.First(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
	if err != nil {
		return fmt.Errorf("getting related record:%w", err)
	}
	if relatedRecord == nil {
		return nil
	}
	err = t.CascadeDelete(key.RelatedDataType, relatedRecord)
	if err != nil {
		return fmt.Errorf("deleting related record: at table '%s' by index '%s', value '%s'",
			key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
	}
	return nil
}

func (t *Txn) archiveChild(archivingTimeStamp int64, archivingHash int64) func(originObjectFieldValue interface{}, key Relation) error {
	return func(parentObjectFiledValue interface{}, key Relation) error {
		relatedRecord, err := t.First(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
		if err != nil {
			return fmt.Errorf("getting related record:%w", err)
		}
		if relatedRecord == nil {
			return nil
		}
		err = t.CascadeArchive(key.RelatedDataType, relatedRecord, archivingTimeStamp, archivingHash)
		if err != nil {
			return fmt.Errorf("archiving related record: at table '%s' by index '%s', value '%s'",
				key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
		}
		return nil
	}
}

func (t *Txn) restoreChild(archivingTimestamp UnixTime, archivingHash int64) func(parentObjectFiledValue interface{}, key Relation) error {
	return func(parentObjectFiledValue interface{}, key Relation) error {
		relatedRecord, err := t.First(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
		if err != nil {
			return fmt.Errorf("getting related record:%w", err)
		}
		if relatedRecord == nil {
			return nil
		}
		a, ok := relatedRecord.(Archivable)
		if !ok {
			return fmt.Errorf("%w related record %#v is not archivable", ErrNotArchivable, relatedRecord)
		}
		if ts, hs := a.ArchiveMarks(); ts != archivingTimestamp || hs != archivingHash {
			return nil
		}
		err = t.CascadeRestore(key.RelatedDataType, relatedRecord)
		if err != nil {
			return fmt.Errorf("restoring related record: at table '%s' by index '%s', value '%s'",
				key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
		}
		return nil
	}
}

func NewMemDB(schema *DBSchema) (*MemDB, error) {
	if err := schema.validate(); err != nil {
		return nil, err
	}
	db, err := memdb.NewMemDB(schema.DBSchema)
	if err != nil {
		return nil, err
	}
	return &MemDB{
		MemDB:  db,
		schema: schema,
	}, nil
}
