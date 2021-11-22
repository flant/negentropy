package memdb

import (
	"fmt"
	"reflect"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

var (
	ErrForeignKey         = fmt.Errorf("foreign key error")
	ErrNotEmptyRelation   = fmt.Errorf("not empty relation error")
	ErrNotArchivable      = fmt.Errorf("not archivable object")
	ErrInvalidSchema      = fmt.Errorf("invalid DBSchema")
	ErrMergeSchema        = fmt.Errorf("merging DBSchema")
	ErrClearingChildSlice = fmt.Errorf("cleaning child StringSlice field")
	ErrNotPtr             = fmt.Errorf("not pointer passed")
)

type MemDB struct {
	*hcmemdb.MemDB

	schema *DBSchema
}

type Txn struct {
	*hcmemdb.Txn

	schema *DBSchema
}

func NewMemDB(schema *DBSchema) (*MemDB, error) {
	if err := schema.Validate(); err != nil {
		return nil, err
	}
	db, err := hcmemdb.NewMemDB(&hcmemdb.DBSchema{Tables: schema.Tables})
	if err != nil {
		return nil, err
	}
	return &MemDB{
		MemDB:  db,
		schema: schema,
	}, nil
}

func (m *MemDB) Txn(write bool) *Txn {
	mTxn := m.MemDB.Txn(write)
	if write {
		mTxn.TrackChanges()
	}
	return &Txn{Txn: mTxn, schema: m.schema}
}

func (t *Txn) Insert(table string, objPtr interface{}) error {
	return t.insert(table, objPtr, 0, 0)
}

// insert provide Insert operation into memdb with checking MandatoryForeignKey,
// insertion successful, if related records exists and aren't archived, or archived with suitable marks
func (t *Txn) insert(table string, objPtr interface{}, allowedArchivingTimeStamp UnixTime, allowedArchivingHash int64) error {
	err := t.checkForeignKeys(table, objPtr, allowedArchivingTimeStamp, allowedArchivingHash)
	if err != nil {
		return fmt.Errorf("insert:%w", err)
	}
	return t.Txn.Insert(table, objPtr)
}

func (t *Txn) Delete(table string, objPtr interface{}) error {
	err := t.checkCascadeDeletesAndCheckingRelations(table, objPtr)
	if err != nil {
		return fmt.Errorf("delete:%w", err)
	}
	err = t.Txn.Delete(table, objPtr)
	if err != nil {
		return fmt.Errorf("delete:%w", err)
	}
	return nil
}

func (t *Txn) CascadeDelete(table string, objPtr interface{}) error {
	err := t.checkCheckingRelations(table, objPtr)
	if err != nil {
		return fmt.Errorf("cascadeDelete:%w", err)
	}
	err = t.processRelations(t.schema.CascadeDeletes[table], objPtr, t.deleteChildren, ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cascadeDelete:%w", err)
	}
	err = t.Txn.Delete(table, objPtr)
	if err != nil {
		return fmt.Errorf("cascadeDelete:%w", err)
	}
	return nil
}

func (t *Txn) Archive(table string, objPtr interface{}, archivingTimeStamp int64, archivingHash int64) error {
	a, ok := objPtr.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, objPtr)
	}
	err := t.checkCascadeDeletesAndCheckingRelations(table, objPtr)
	if err != nil {
		return fmt.Errorf("archive:%w", err)
	}
	a.Archive(archivingTimeStamp, archivingHash)
	err = t.Insert(table, objPtr)
	if err != nil {
		return fmt.Errorf("archive:%w", err)
	}
	return nil
}

func (t *Txn) CascadeArchive(table string, objPtr interface{}, archivingTimeStamp int64, archivingHash int64) error {
	a, ok := objPtr.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, objPtr)
	}
	err := t.checkCheckingRelations(table, objPtr)
	if err != nil {
		return fmt.Errorf("cascadeArchive:%w", err)
	}
	err = t.processRelations(t.schema.CascadeDeletes[table], objPtr, t.archiveChildren(archivingTimeStamp, archivingHash), ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cascadeArchive:%w", err)
	}
	a.Archive(archivingTimeStamp, archivingHash)
	err = t.Insert(table, objPtr)
	if err != nil {
		return fmt.Errorf("cascadeArchive:%w", err)
	}
	return nil
}

func (t *Txn) Restore(table string, objPtr interface{}) error {
	a, ok := objPtr.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, objPtr)
	}
	a.Restore()
	err := t.Insert(table, objPtr)
	if err != nil {
		return fmt.Errorf("restore:%w", err)
	}
	return nil
}

func (t *Txn) CascadeRestore(table string, objPtr interface{}) error {
	a, ok := objPtr.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, objPtr)
	}
	timeStamp, archivingHash := a.ArchiveMarks()
	a.Restore()
	err := t.insert(table, objPtr, timeStamp, archivingHash)
	if err != nil {
		return fmt.Errorf("cascadeRestore:%w", err)
	}
	err = t.processRelations(t.schema.CascadeDeletes[table], objPtr, t.restoreChildren(timeStamp, archivingHash), ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cascadeRestore:%w", err)
	}
	return nil
}

func (t *Txn) checkForeignKeys(table string, objPtr interface{}, allowedArchivingTimeStamp UnixTime, allowedArchivingHash int64) error {
	keys := t.schema.MandatoryForeignKeys[table]
	return t.processRelations(keys, objPtr, t.checkForeignKey(allowedArchivingTimeStamp, allowedArchivingHash), ErrForeignKey)
}

// checkForeignKey supports Slice as a field type
func (t *Txn) checkForeignKey(allowedArchivingTimeStamp UnixTime, allowedArchivingHash int64) func(checkedFieldValue interface{}, key Relation) error {
	return func(checkedFieldValue interface{}, key Relation) error {
		switch reflect.TypeOf(checkedFieldValue).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(checkedFieldValue)
			for i := 0; i < s.Len(); i++ {
				err := t.checkSingleValueOfForeignKey(s.Index(i).Interface(), key, allowedArchivingTimeStamp, allowedArchivingHash)
				if err != nil {
					return err
				}
			}
			return nil
		default:
			return t.checkSingleValueOfForeignKey(checkedFieldValue, key, allowedArchivingTimeStamp, allowedArchivingHash)
		}
	}
}

// singleCheckedFieldValue should not be pointer
func (t *Txn) checkSingleValueOfForeignKey(singleCheckedFieldValue interface{}, key Relation,
	allowedArchivingTimeStamp UnixTime, allowedArchivingHash int64) error {
	var err error
	if key.BuildRelatedCustomType != nil {
		singleCheckedFieldValue, err = key.BuildRelatedCustomType(singleCheckedFieldValue)
		if err != nil {
			return fmt.Errorf("mapping: %w", err)
		}
	}

	relatedRecord, err := t.First(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, singleCheckedFieldValue)
	if err != nil {
		return fmt.Errorf("getting related record:%w", err)
	}
	if relatedRecord == nil {
		return fmt.Errorf("FK violation: %q not found at table %q at index %q",
			singleCheckedFieldValue, key.RelatedDataType, key.RelatedDataTypeFieldIndexName)
	}
	a, ok := relatedRecord.(Archivable)
	if !ok {
		if allowedArchivingTimeStamp == 0 && allowedArchivingHash == 0 {
			return nil // related record is not archivable, exists, an no need to check
		} else {
			return fmt.Errorf("%w related record %#v is not archivable", ErrNotArchivable, relatedRecord)
		}
	}
	s, h := a.ArchiveMarks()
	if a.Archived() && (s != allowedArchivingTimeStamp || h != allowedArchivingHash) {
		return fmt.Errorf("FK violation: %q not found at table %q at index %q",
			singleCheckedFieldValue, key.RelatedDataType, key.RelatedDataTypeFieldIndexName)
	}
	return nil
}

func (t *Txn) checkCascadeDeletesAndCheckingRelations(table string, objPtr interface{}) error {
	rels := append(t.schema.CascadeDeletes[table], t.schema.CheckingRelations[table]...) //nolint:gocritic
	return t.processRelations(rels, objPtr, t.checkRelationShouldBeEmpty, ErrNotEmptyRelation)
}

func (t *Txn) checkCheckingRelations(table string, objPtr interface{}) error {
	rels := t.schema.CheckingRelations[table]
	return t.processRelations(rels, objPtr, t.checkRelationShouldBeEmpty, ErrNotEmptyRelation)
}

// implement main loop checking relations
// for each relation from relations, will be executed relationHandler
func (t *Txn) processRelations(relations []Relation, objPtr interface{},
	relationHandler func(originObjectFieldValue interface{}, key Relation) error,
	relationHandlerError error) error {
	valueIface := reflect.ValueOf(objPtr)
	if valueIface.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("obj `%s` is not ptr", valueIface.Type())
	}
	errorCollector := utils.ErrorCollector{}
	for _, key := range relations {
		field := valueIface.Elem().FieldByName(key.OriginalDataTypeFieldName)
		if !field.IsValid() {
			return fmt.Errorf("obj `%s` does not have the field `%s`", valueIface.Type(), key.OriginalDataTypeFieldName)
		}
		checkedFieldValue := field.Interface()
		if err := relationHandler(checkedFieldValue, key); err != nil {
			errorCollector.Collect(err)
		}
	}
	if !errorCollector.Empty() {
		return fmt.Errorf("%w:%s", relationHandlerError, errorCollector.Error())
	}
	return nil
}

// checkedFieldValue should not be pointer
func (t *Txn) checkRelationShouldBeEmpty(checkedFieldValue interface{}, key Relation) error {
	var err error
	if key.BuildRelatedCustomType != nil {
		checkedFieldValue, err = key.BuildRelatedCustomType(checkedFieldValue)
		if err != nil {
			return fmt.Errorf("using BuildRelatedCustomType:%w", err)
		}
	}
	relatedRecord, err := t.First(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, checkedFieldValue)
	if err != nil {
		return fmt.Errorf("getting related record:%w", err)
	}
	if relatedRecord != nil {
		return fmt.Errorf("relation should be empty: %q found at table %q by index %q",
			checkedFieldValue, key.RelatedDataType, key.RelatedDataTypeFieldIndexName)
	}
	return nil
}

// parentObjectFiledValue should not be pointer
func (t *Txn) deleteChildren(parentObjectFiledValue interface{}, key Relation) error {
	if key.indexIsSliceFieldIndex {
		return nil
	}
	if key.BuildRelatedCustomType != nil {
		// TODO CleanChildrenSliceIndexes not implemented yet for CustomTypeFieldIndexer
		return fmt.Errorf("CleanChildrenSliceIndexes not implemented yet for CustomTypeFieldIndexer")
	}
	iter, err := t.Get(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
	if err != nil {
		return fmt.Errorf("getting related record:%w", err)
	}
	for {
		relatedRecord := iter.Next()
		if relatedRecord == nil {
			break
		}
		relatedRecord, err := t.First(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
		if err != nil {
			return fmt.Errorf("getting related record:%w", err)
		}
		err = t.CascadeDelete(key.RelatedDataType, relatedRecord)
		if err != nil {
			return fmt.Errorf("deleting related record: at table %q by index %q, value %q",
				key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
		}
	}
	return nil
}

// originObjectFieldValue should not be pointer
func (t *Txn) archiveChildren(archivingTimeStamp int64, archivingHash int64) func(originObjectFieldValue interface{}, key Relation) error {
	return func(parentObjectFiledValue interface{}, key Relation) error {
		if key.BuildRelatedCustomType != nil {
			// TODO CleanChildrenSliceIndexes not implemented yet for CustomTypeFieldIndexer
			return fmt.Errorf("CleanChildrenSliceIndexes not implemented yet for CustomTypeFieldIndexer")
		}

		if key.indexIsSliceFieldIndex {
			return nil
		}
		iter, err := t.Get(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
		if err != nil {
			return fmt.Errorf("getting related record:%w", err)
		}
		for {
			relatedRecord := iter.Next()
			if relatedRecord == nil {
				break
			}
			a, ok := relatedRecord.(Archivable)
			if !ok {
				return fmt.Errorf("%w related record %#v is not archivable", ErrNotArchivable, relatedRecord)
			}
			if a.Archived() {
				continue
			}
			err = t.CascadeArchive(key.RelatedDataType, relatedRecord, archivingTimeStamp, archivingHash)
			if err != nil {
				return fmt.Errorf("archiving related record: at table %q by index %q, value %q",
					key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
			}
		}
		return nil
	}
}

// parentObjectFiledValue should not be pointer
func (t *Txn) restoreChildren(archivingTimestamp UnixTime, archivingHash int64) func(parentObjectFiledValue interface{}, key Relation) error {
	return func(parentObjectFiledValue interface{}, key Relation) error {
		if key.BuildRelatedCustomType != nil {
			// TODO CleanChildrenSliceIndexes not implemented yet for CustomTypeFieldIndexer
			return fmt.Errorf("CleanChildrenSliceIndexes not implemented yet for CustomTypeFieldIndexer")
		}
		if key.indexIsSliceFieldIndex {
			return nil
		}
		iter, err := t.Get(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
		if err != nil {
			return fmt.Errorf("getting related record:%w", err)
		}
		for {
			relatedRecord := iter.Next()
			if relatedRecord == nil {
				break
			}
			a, ok := relatedRecord.(Archivable)
			if !ok {
				return fmt.Errorf("%w related record %#v is not archivable", ErrNotArchivable, relatedRecord)
			}
			if ts, hs := a.ArchiveMarks(); ts != archivingTimestamp || hs != archivingHash {
				continue
			}
			err = t.CascadeRestore(key.RelatedDataType, relatedRecord)
			if err != nil {
				return fmt.Errorf("restoring related record: at table %q by index %q, value %q",
					key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
			}
		}
		return nil
	}
}

// CleanChildrenSliceIndexes remove link to obj from children slice fields
func (t *Txn) CleanChildrenSliceIndexes(table string, objPtr interface{}) error {
	// TODO is it a objPtr?
	cleanChildrenSlicesHandler := func(parentObjectFieldValue interface{}, key Relation) error {
		if key.BuildRelatedCustomType != nil {
			return fmt.Errorf("CleanChildrenSliceIndexes not implemented yet for CustomTypeFieldIndexer")
		}
		var parentObjectFieldValueStr string
		var ok bool
		if !key.indexIsSliceFieldIndex {
			return nil
		}
		if parentObjectFieldValueStr, ok = parentObjectFieldValue.(string); !ok {
			return fmt.Errorf("wrong type of parentObjectFieldValue:%T", parentObjectFieldValue)
		}
		iter, err := t.Get(key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFieldValue)
		if err != nil {
			return fmt.Errorf("getting related record:%w", err)
		}
		for {
			relatedRecord := iter.Next()
			if relatedRecord == nil {
				break
			}
			// Получить имя поля
			index, ok := t.schema.Tables[key.RelatedDataType].Indexes[key.RelatedDataTypeFieldIndexName].Indexer.(*hcmemdb.StringSliceFieldIndex)
			if !ok {
				return fmt.Errorf("wrong type of index, for relation %#v, of table %q", key, table)
			}
			valueIface := reflect.ValueOf(relatedRecord)
			fieldValue := valueIface.Elem().FieldByName(index.Field).Interface()
			vals, ok := fieldValue.([]string)
			if !ok {
				return fmt.Errorf("wrong type of related record field %q: need []string, actual %T", index.Field, fieldValue)
			}
			newVals := []string{}
			for _, v := range vals {
				if v != parentObjectFieldValueStr {
					newVals = append(newVals, v)
				}
			}
			valueIface.Elem().FieldByName(index.Field).Set(reflect.ValueOf(newVals))
			err := t.insert(key.RelatedDataType, relatedRecord, 0, 0)
			if err != nil {
				return nil
			}
		}
		return nil
	}

	err := t.processRelations(t.schema.CascadeDeletes[table], objPtr, cleanChildrenSlicesHandler, ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cleanChildrenSliceIndexes:%w", err)
	}
	return nil
}

func checkPtrAndReturnIndirect(objPtr interface{}) (obj interface{}, err error) {
	valueIface := reflect.ValueOf(objPtr)
	if valueIface.Type().Kind() != reflect.Ptr {
		return nil, fmt.Errorf("%w:%T", ErrNotPtr, objPtr)
	}
	return reflect.Indirect(valueIface).Interface(), nil
}
