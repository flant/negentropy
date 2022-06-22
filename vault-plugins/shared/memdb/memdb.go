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
	ErrUniqueConstraint   = fmt.Errorf("fail unique constraint")
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
	return t.insert(table, objPtr, ActiveRecordMark)
}

// insert provide Insert operation into memdb with checking MandatoryForeignKey,
// insertion successful, if related records exists and aren't archived, or archived with suitable marks
func (t *Txn) insert(table string, objPtr interface{}, allowedArchiveMark ArchiveMark) error {
	err := t.checkUniqueConstraints(table, objPtr)
	if err != nil {
		return fmt.Errorf("insert %#v: %w", objPtr, err)
	}
	err = t.checkForeignKeys(table, objPtr, allowedArchiveMark)
	if err != nil {
		return fmt.Errorf("insert %#v: %w", objPtr, err)
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

func (t *Txn) Archive(table string, objPtr interface{}, archiveMark ArchiveMark) error {
	a, ok := objPtr.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, objPtr)
	}
	err := t.checkCascadeDeletesAndCheckingRelations(table, objPtr)
	if err != nil {
		return fmt.Errorf("archive:%w", err)
	}
	a.Archive(archiveMark)
	err = t.insert(table, objPtr, archiveMark)
	if err != nil {
		return fmt.Errorf("archive:%w", err)
	}
	return nil
}

func (t *Txn) CascadeArchive(table string, objPtr interface{}, archiveMark ArchiveMark) error {
	a, ok := objPtr.(Archivable)
	if !ok {
		return fmt.Errorf("%w:%#v", ErrNotArchivable, objPtr)
	}
	err := t.checkCheckingRelations(table, objPtr)
	if err != nil {
		return fmt.Errorf("cascadeArchive:%w", err)
	}
	err = t.processRelations(t.schema.CascadeDeletes[table], objPtr, t.archiveChildren(archiveMark), ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cascadeArchive:%w", err)
	}
	a.Archive(archiveMark)
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
	archiveMark := a.GetArchiveMark()
	a.Restore()
	err := t.insert(table, objPtr, archiveMark)
	if err != nil {
		return fmt.Errorf("cascadeRestore:%w", err)
	}
	err = t.processRelations(t.schema.CascadeDeletes[table], objPtr, t.restoreChildren(archiveMark), ErrNotEmptyRelation)
	if err != nil {
		return fmt.Errorf("cascadeRestore:%w", err)
	}
	return nil
}

func (t *Txn) checkForeignKeys(table string, objPtr interface{}, allowedArchiveMark ArchiveMark) error {
	keys := t.schema.MandatoryForeignKeys[table]
	return t.processRelations(keys, objPtr, t.checkForeignKey(allowedArchiveMark), ErrForeignKey)
}

// checkForeignKey supports Slice as a field type
func (t *Txn) checkForeignKey(allowedArchiveMark ArchiveMark) func(checkedFieldValue interface{}, key Relation) error {
	return func(checkedFieldValue interface{}, key Relation) error {
		switch reflect.TypeOf(checkedFieldValue).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(checkedFieldValue)
			for i := 0; i < s.Len(); i++ {
				err := t.checkSingleValueOfForeignKey(s.Index(i).Interface(), key, allowedArchiveMark)
				if err != nil {
					return err
				}
			}
			return nil
		default:
			return t.checkSingleValueOfForeignKey(checkedFieldValue, key, allowedArchiveMark)
		}
	}
}

// singleCheckedFieldValue should not be pointer
func (t *Txn) checkSingleValueOfForeignKey(singleCheckedFieldValue interface{}, key Relation,
	allowedArchiveMark ArchiveMark) error {
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
		if ActiveRecordMark.Equals(allowedArchiveMark) {
			return nil // related record is not archivable, exists, an no need to check
		} else {
			return fmt.Errorf("%w related record %#v is not archivable", ErrNotArchivable, relatedRecord)
		}
	}
	if a.Archived() && !a.Equals(allowedArchiveMark) {
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
	if relatedRecord == nil {
		return nil
	}
	a, ok := relatedRecord.(Archivable)
	if !ok {
		return fmt.Errorf("got not archivable object: by key value %q found at table %q by index %q",
			checkedFieldValue, key.RelatedDataType, key.RelatedDataTypeFieldIndexName)
	}
	if a.NotArchived() {
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
func (t *Txn) archiveChildren(archiveMark ArchiveMark) func(originObjectFieldValue interface{}, key Relation) error {
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
			err = t.CascadeArchive(key.RelatedDataType, relatedRecord, archiveMark)
			if err != nil {
				return fmt.Errorf("archiving related record: at table %q by index %q, value %q",
					key.RelatedDataType, key.RelatedDataTypeFieldIndexName, parentObjectFiledValue)
			}
		}
		return nil
	}
}

// parentObjectFiledValue should not be pointer
func (t *Txn) restoreChildren(allowedArchiveMark ArchiveMark) func(parentObjectFiledValue interface{}, key Relation) error {
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
			if !a.Equals(allowedArchiveMark) {
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
			err := t.insert(key.RelatedDataType, relatedRecord, ActiveRecordMark)
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

type storable interface {
	ObjType() string
	ObjId() string
}

// check uniqueConstraints among other unarchived records
func (t *Txn) checkUniqueConstraints(table string, objPtr interface{}) error {
	if a, isArchivable := objPtr.(Archivable); isArchivable && a.Archived() { // check only valid insertion
		return nil
	}
	objID := ""
	if s, isStorable := objPtr.(storable); isStorable {
		objID = s.ObjId()
	}
	var indexesToCheck []*hcmemdb.IndexSchema
	for _, idxName := range t.schema.UniqueConstraints[table] {
		indexesToCheck = append(indexesToCheck, t.schema.Tables[table].Indexes[idxName])
	}
	if len(indexesToCheck) != 0 {
		for _, idx := range indexesToCheck {
			vals, err := collectValsForIndexes(objPtr, idx.Indexer)
			if err != nil {
				return fmt.Errorf("collecting vals for index %s at table %s: %w", idx.Name, table, err)
			}
			err = t.checkIdxIsEmpty(table, idx.Name, vals, objID)
			if err != nil {
				return fmt.Errorf("checkUniqueConstraints: %w", err)
			}
		}
	}
	return nil
}

func (t *Txn) checkIdxIsEmpty(table string, idxName string, vals []interface{}, savedObjID string) error {
	iter, err := t.Get(table, idxName, vals...)
	if err != nil {
		return fmt.Errorf("checkIdxIsEmpty, index: %q at table %q: %w", idxName, table, err)
	}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		if s, isStorable := raw.(storable); isStorable {
			if s.ObjId() == savedObjID { // it is replaced obj, skip
				continue
			}
		}
		a, isArchivable := raw.(Archivable)
		if !isArchivable || a.NotArchived() {
			return fmt.Errorf("%w: %q at table %q", ErrUniqueConstraint, idxName, table)
		}
	}
	return nil
}

func collectValsForIndexes(objPtr interface{}, indexes ...hcmemdb.Indexer) ([]interface{}, error) {
	var vals []interface{}
	for _, idx := range indexes {
		singleFieldName := ""
		switch t := idx.(type) {
		case *hcmemdb.UUIDFieldIndex:
			singleFieldName = t.Field
		case *hcmemdb.StringFieldIndex:
			singleFieldName = t.Field
		case *hcmemdb.CompoundIndex:
			extraVals, err := collectValsForIndexes(objPtr, t.Indexes...)
			if err != nil {
				return nil, err
			}
			vals = append(vals, extraVals...)
		default:
			return nil, fmt.Errorf("index type %t is not supported for unique constarain", idx)
		}
		if singleFieldName != "" {
			valueIface := reflect.ValueOf(objPtr)
			fieldValue := valueIface.Elem().FieldByName(singleFieldName).Interface()
			vals = append(vals, fieldValue)
		}
	}
	return vals, nil
}

func checkPtrAndReturnIndirect(objPtr interface{}) (obj interface{}, err error) {
	valueIface := reflect.ValueOf(objPtr)
	if valueIface.Type().Kind() != reflect.Ptr {
		return nil, fmt.Errorf("%w:%T", ErrNotPtr, objPtr)
	}
	return reflect.Indirect(valueIface).Interface(), nil
}
