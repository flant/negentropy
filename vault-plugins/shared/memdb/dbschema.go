package memdb

import (
	"fmt"
	"strings"

	hcmemdb "github.com/hashicorp/go-memdb"
)

// PK is a mandatory index for all tables at hc/go-memdb
const PK = "id"

type (
	// UnixTime used as timestamp at Timestamp
	UnixTime = int64

	// TableSchema synonym for replacing original type at code
	TableSchema = hcmemdb.TableSchema
)

type Relation struct {
	OriginalDataTypeFieldName fieldName
	RelatedDataType           DataType
	// Only StringFieldIndex or StringSliceFieldIndex
	RelatedDataTypeFieldIndexName IndexName
	// mark as StringSliceFieldIndex or CustomTypeSliceFieldIndex
	indexIsSliceFieldIndex bool
	// buildRelatedCustomType build from fieldValue object for using as an arg for index search
	BuildRelatedCustomType func(originalFieldValue interface{}) (customTypeValue interface{}, err error)
}

func (r *Relation) MapKey() RelationKey {
	return RelationKey{
		OriginalDataTypeFieldName:     r.OriginalDataTypeFieldName,
		RelatedDataType:               r.RelatedDataType,
		RelatedDataTypeFieldIndexName: r.RelatedDataTypeFieldIndexName,
	}
}

func (r *Relation) validateBuildRelatedCustomType(shouldBeNil bool) error {
	if shouldBeNil {
		if r.BuildRelatedCustomType != nil {
			return fmt.Errorf("index named %q at table %q, passed as relation to field %q should not have BuildRelatedCustomType",
				r.RelatedDataTypeFieldIndexName, r.RelatedDataType, r.OriginalDataTypeFieldName)
		}
		return nil
	} else if r.BuildRelatedCustomType == nil {
		return fmt.Errorf("index named %q at table %q, passed as relation to field %q of table should have BuildRelatedCustomType",
			r.RelatedDataTypeFieldIndexName, r.RelatedDataType, r.OriginalDataTypeFieldName)
	}
	return nil
}

// RelationKey represents Relation as struct can be used as a map key
type RelationKey struct {
	OriginalDataTypeFieldName     fieldName
	RelatedDataType               DataType
	RelatedDataTypeFieldIndexName IndexName
}

type DBSchema struct {
	Tables map[string]*TableSchema
	// check at Insert
	// prohibited to use the same DataType as map key and as value in Relation.RelatedDataType
	MandatoryForeignKeys map[DataType][]Relation
	// use at CascadeDelete
	// check at Delete, deleting fails if any of relation is not empty
	// prohibited to use the same DataType as map key and as value in Relation.RelatedDataType
	CascadeDeletes map[DataType][]Relation
	// check at CascadeDelete and Delete, deleting fails if any of relation is not empty
	// prohibited to place the same Relation into CascadeDeletes and CheckingRelations
	// prohibited to use the same DataType as map key and as value in Relation.RelatedDataType
	CheckingRelations map[DataType][]Relation
	// check at Insert and Restore:
	// ResultIterator should be empty or contains only Archived objects or object self
	UniqueConstraints map[DataType][]IndexName
}

type (
	DataType  = string
	fieldName = string
	IndexName = string
)

func (s *DBSchema) Validate() error {
	if err := (&hcmemdb.DBSchema{Tables: s.Tables}).Validate(); err != nil {
		return fmt.Errorf("%w:%s", ErrInvalidSchema, err)
	}
	if err := s.validateExistenceIndexes(); err != nil {
		return fmt.Errorf("%w:%s", ErrInvalidSchema, err)
	}
	if err := validateForeignKeys(s.MandatoryForeignKeys); err != nil {
		return fmt.Errorf("%w:%s", ErrInvalidSchema, err)
	}
	allChildRelations, err := s.validateUniquenessChildRelations()
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
// 1) absence the same DataType as key and RelatedDataType
// 2) absence of cyclic dependencies
// 3) only 'id' as RelatedDataTypeFieldIndexName
func validateForeignKeys(fks map[DataType][]Relation) error {
	type childDataType = DataType
	rels := map[DataType]map[RelationKey]struct{}{}
	for d, keys := range fks {
		ks, ok := rels[d]
		if !ok {
			ks = map[RelationKey]struct{}{}
		}
		for _, key := range keys {
			if key.RelatedDataTypeFieldIndexName != PK {
				return fmt.Errorf("invalid RelatedDataTypeFieldIndexName:%s in FK:%#v of table %s",
					key.RelatedDataTypeFieldIndexName, key, d)
			}
			ks[key.MapKey()] = struct{}{}
		}
		rels[d] = ks
	}
	for d := range rels {
		err := validateCyclic(d, rels)
		if err != nil {
			return fmt.Errorf("сyclic dependency: %s",
				err.Error())
		}
	}
	return nil
}

// dependency is a one level of relations of evaluated DataType
// used as element of recursive stack
type dependency struct {
	// processed datatype, at lvl=0, it is  topDataType
	parentType DataType
	// all direct children of parentType
	directChildrenDataTypes []DataType
	// used as a pointer to currently processed at next level parentType
	currentChildIdx int
}

// validateCyclic checks absence of cyclic dependencies between tables
func validateCyclic(topDataType DataType, rels map[DataType]map[RelationKey]struct{}) error {
	// findChildrenDataTypes extracts from rels all direct children relations for parentDataType
	findChildrenDataTypes := func(parentDataType DataType, rels map[DataType]map[RelationKey]struct{}) []DataType {
		mapResult := map[DataType]struct{}{}
		for r := range rels[parentDataType] {
			// allow self-links
			if r.RelatedDataType != parentDataType {
				mapResult[r.RelatedDataType] = struct{}{}
			}
		}
		result := make([]DataType, 0, len(mapResult))
		for r := range mapResult {
			result = append(result, r)
		}
		return result
	}

	deps := make([]dependency, len(rels)+1)
	deps[0] = dependency{
		parentType:              topDataType,
		currentChildIdx:         0,
		directChildrenDataTypes: findChildrenDataTypes(topDataType, rels),
	}
	lvl := 0
	for lvl != 0 || deps[0].currentChildIdx < len(deps[0].directChildrenDataTypes) {
		curIdx := deps[lvl].currentChildIdx
		switch {
		case curIdx >= len(deps[lvl].directChildrenDataTypes):
			lvl--
			deps[lvl].currentChildIdx++
			continue
		case deps[lvl].directChildrenDataTypes[curIdx] == topDataType:
			return fmt.Errorf("dependencies chain:%s", formatChain(deps[0:lvl+1]))
		case len(rels[deps[lvl].directChildrenDataTypes[curIdx]]) > 0:
			curType := deps[lvl].directChildrenDataTypes[curIdx]
			lvl++
			deps[lvl] = dependency{
				parentType:              curType,
				currentChildIdx:         0,
				directChildrenDataTypes: findChildrenDataTypes(curType, rels),
			}
		default:
			deps[lvl].currentChildIdx++
		}
	}
	return nil
}

func formatChain(deps []dependency) string {
	stringBuilder := strings.Builder{}
	for _, d := range deps {
		if stringBuilder.Len() != 0 {
			stringBuilder.WriteString("=>")
		}
		stringBuilder.WriteString(d.parentType)
	}
	stringBuilder.WriteString("=>" + deps[0].parentType)
	return stringBuilder.String()
}

// validateExistenceIndexes check existenceIndexes at tables, and fill indexIsSliceFieldIndex
func (s *DBSchema) validateExistenceIndexes() error {
	checkRelation := func(mapRels *map[DataType][]Relation, childrenRelations bool) error {
		tables := s.Tables
		for dt, rs := range *mapRels {
			for i := 0; i < len(rs); i++ {
				relation := &rs[i]
				r, err := verifyIndex(dt, tables, relation, childrenRelations)
				if err != nil {
					return err
				}
				rs[i] = *r
			}
			(*mapRels)[dt] = rs
		}
		return nil
	}

	chlidrenRels := []bool{false, true, true}
	for i, rs := range []*map[DataType][]Relation{&s.MandatoryForeignKeys, &s.CascadeDeletes, &s.CheckingRelations} {
		if err := checkRelation(rs, chlidrenRels[i]); err != nil {
			return err
		}
	}
	for tableName, idxNames := range s.UniqueConstraints {
		table, found := s.Tables[tableName]
		if !found {
			return fmt.Errorf("%w: wrong table name at unique constrains: %q", ErrInvalidSchema, tableName)
		}
		for _, idxName := range idxNames {
			if idxName == PK {
				return fmt.Errorf("%w: unique_constrains should not be PK", ErrInvalidSchema)
			}
			idx, found := table.Indexes[idxName]
			if !found {
				return fmt.Errorf("%w: not found index %q passed as unique_constrain for %q table", ErrInvalidSchema, idxName, tableName)
			}
			if idx.Unique {
				return fmt.Errorf("%w: allow to use at unique_constrains only 'Unique=false' indexes: passed %q index at %q table", ErrInvalidSchema, idxName, tableName)
			}
		}
	}

	return nil
}

func verifyIndex(dt DataType, tables map[string]*TableSchema, r *Relation, childrenRelations bool) (*Relation, error) {
	if ts, ok := tables[r.RelatedDataType]; !ok {
		return nil, fmt.Errorf("table %s is absent in DBSchema", r.RelatedDataType)
	} else {
		if index, ok := ts.Indexes[r.RelatedDataTypeFieldIndexName]; ok {
			switch index.Indexer.(type) {
			case *hcmemdb.StringFieldIndex:
				if err := r.validateBuildRelatedCustomType(true); err != nil && childrenRelations {
					return nil, fmt.Errorf("table %s:%w", dt, err)
				}
			case *hcmemdb.UUIDFieldIndex:
				if err := r.validateBuildRelatedCustomType(true); err != nil && childrenRelations {
					return nil, fmt.Errorf("table %s:%w", dt, err)
				}
			case *CustomTypeFieldIndexer:
				if err := r.validateBuildRelatedCustomType(false); err != nil && childrenRelations {
					return nil, fmt.Errorf("table %s:%w", dt, err)
				}
			case *hcmemdb.StringSliceFieldIndex:
				if err := r.validateBuildRelatedCustomType(true); err != nil && childrenRelations {
					return nil, fmt.Errorf("table %s:%w", dt, err)
				}
				r.indexIsSliceFieldIndex = true
			case *CustomTypeSliceFieldIndexer:
				if err := r.validateBuildRelatedCustomType(false); err != nil && childrenRelations {
					return nil, fmt.Errorf("table %s:%w", dt, err)
				}
				r.indexIsSliceFieldIndex = true
			default:
				return nil, fmt.Errorf("index named %q at table %q, passed as relation to field %q of table "+
					"%q has inapropriate type (allowed: StringFieldIndex,UUIDFieldIndex, StringSliceFieldIndex, "+
					"CustomTypeSliceFieldIndexer, CustomTypeFieldIndexer)",
					r.RelatedDataTypeFieldIndexName, r.RelatedDataType, r.OriginalDataTypeFieldName, dt)
			}
		} else {
			return nil, fmt.Errorf("index named %q not found at table %q, passed as relation to field %q of table %q",
				r.RelatedDataTypeFieldIndexName, r.RelatedDataType, r.OriginalDataTypeFieldName, dt)
		}
	}
	return r, nil
}

// validateUniquenessChildRelations checks uniqueness for CascadeDeletes and CheckingRelations
// returns united map of relations
func (s *DBSchema) validateUniquenessChildRelations() (map[DataType]map[RelationKey]struct{}, error) {
	allRels := map[DataType]map[RelationKey]struct{}{}
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
func checkRsMapForRepeating(allRels map[DataType]map[RelationKey]struct{},
	rsMap map[DataType][]Relation) (map[DataType]map[RelationKey]struct{}, error) {
	for d, rs := range rsMap {
		if rels, ok := allRels[d]; ok {
			for _, r := range rs {
				if _, rep := rels[r.MapKey()]; rep {
					return nil, fmt.Errorf("relation %#v is repeated for %s DataType", r, d)
				}
				rels[r.MapKey()] = struct{}{}
			}
		} else {
			rels := map[RelationKey]struct{}{}
			for _, r := range rs {
				rels[r.MapKey()] = struct{}{}
			}
			allRels[d] = rels
		}
	}
	return allRels, nil
}

func MergeDBSchemasAndValidate(schemas ...*DBSchema) (*DBSchema, error) {
	return MergeDBSchemas(true, schemas...)
}

func MergeDBSchemas(validate bool, schemas ...*DBSchema) (*DBSchema, error) {
	tables := map[string]*hcmemdb.TableSchema{}
	uniqueConstrains := map[DataType][]IndexName{}

	for i := range schemas {
		for tableName, table := range schemas[i].Tables {
			if _, found := tables[tableName]; found {
				return nil, fmt.Errorf("%w: table %q already there", ErrMergeSchema, tableName)
			}
			tables[tableName] = table
		}
		for tableName, IndexNames := range schemas[i].UniqueConstraints {
			if _, found := uniqueConstrains[tableName]; found {
				return nil, fmt.Errorf("%w: unique_constrains %q already there", ErrMergeSchema, tableName)
			}
			for _, idxName := range IndexNames {
				if idxName == PK {
					return nil, fmt.Errorf("%w: unique_constrains should not be PK", ErrInvalidSchema)
				}
				idx := tables[tableName].Indexes[idxName]
				if idx.Unique {
					return nil, fmt.Errorf("%w: allow to use at unique_constrains only not 'Unique=true' indexes: passed %s index at %s table", ErrInvalidSchema, idxName, tableName)
				}
			}
			uniqueConstrains[tableName] = IndexNames
		}
	}

	type mapRelations = map[DataType][]Relation

	mergeRelationFunc := func(getRelationsFunc func(*DBSchema) mapRelations, schemas ...*DBSchema) map[DataType][]Relation {
		allRels := map[DataType][]Relation{}
		for _, schema := range schemas {
			for name, rels := range getRelationsFunc(schema) {
				if prevRels, found := allRels[name]; found {
					rels = append(prevRels, rels...)
				}
				allRels[name] = rels
			}
		}
		return allRels
	}

	result := DBSchema{
		Tables:               tables,
		MandatoryForeignKeys: mergeRelationFunc(func(s *DBSchema) mapRelations { return s.MandatoryForeignKeys }, schemas...),
		CascadeDeletes:       mergeRelationFunc(func(s *DBSchema) mapRelations { return s.CascadeDeletes }, schemas...),
		CheckingRelations:    mergeRelationFunc(func(s *DBSchema) mapRelations { return s.CheckingRelations }, schemas...),
		UniqueConstraints:    uniqueConstrains,
	}

	if validate {
		err := result.Validate()
		if err != nil {
			return nil, fmt.Errorf("%w:%s", ErrMergeSchema, err.Error())
		}
	}
	return &result, nil
}

func DropRelations(schema *DBSchema) *DBSchema {
	schema.MandatoryForeignKeys = nil
	schema.CascadeDeletes = nil
	schema.CheckingRelations = nil
	return schema
}
