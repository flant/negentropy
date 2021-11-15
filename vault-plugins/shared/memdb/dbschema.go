package memdb

import (
	"fmt"
	"strings"

	hcmemdb "github.com/hashicorp/go-memdb"
)

// PK is a mandatory index for all tables at hc/go-memdb
const PK = "id"

type (
	// UnixTime used as timestamp at ArchivingTimestamp
	UnixTime = int64

	// TableSchema synonym for replacing original type at code
	TableSchema = hcmemdb.TableSchema
)

type Relation struct {
	OriginalDataTypeFieldName fieldName
	RelatedDataType           dataType
	// Only StringFieldIndex or StringSliceFieldIndex
	RelatedDataTypeFieldIndexName indexName
	// mark as StringSliceFieldIndex
	indexIsSliceFieldIndex bool
}

type DBSchema struct {
	Tables map[string]*TableSchema
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

type (
	dataType  = string
	fieldName = string
	indexName = string
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
			// allow self-links
			if r.RelatedDataType != parentDataType {
				mapResult[r.RelatedDataType] = struct{}{}
			}
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

// validateExistenceIndexes check existenceIndexes at tables, and fill indexIsSliceFieldIndex
func (s *DBSchema) validateExistenceIndexes() error {
	checkRelation := func(mapRels *map[dataType][]Relation) error {
		tables := s.Tables
		for dt, rs := range *mapRels {
			for i := 0; i < len(rs); i++ {
				r := rs[i]
				if ts, ok := tables[r.RelatedDataType]; !ok {
					return fmt.Errorf("table %s is absent in DBSchema", dt)
				} else {
					if index, ok := ts.Indexes[r.RelatedDataTypeFieldIndexName]; ok {
						switch index.Indexer.(type) {
						case *hcmemdb.StringFieldIndex:
						case *hcmemdb.UUIDFieldIndex:
						case *hcmemdb.StringSliceFieldIndex:
							r.indexIsSliceFieldIndex = true
							rs[i] = r
						default:
							return fmt.Errorf("index named %q at table %q, passed as relation to field %q of table "+
								"%q has inapropriate type (allowed: StringFieldIndex,UUIDFieldIndex, StringSliceFieldIndex)",
								r.RelatedDataTypeFieldIndexName, r.RelatedDataType, r.OriginalDataTypeFieldName, dt)
						}
					} else {
						return fmt.Errorf("index named %q not found at table %q, passed as relation to field %q of table %q",
							r.RelatedDataTypeFieldIndexName, r.RelatedDataType, r.OriginalDataTypeFieldName, dt)
					}
				}
			}
			(*mapRels)[dt] = rs
		}
		return nil
	}
	for _, rs := range []*map[dataType][]Relation{&s.MandatoryForeignKeys, &s.CascadeDeletes, &s.CheckingRelations} {
		if err := checkRelation(rs); err != nil {
			return err
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

func MergeDBSchemas(schemas ...*DBSchema) (*DBSchema, error) {
	tables := map[string]*hcmemdb.TableSchema{}

	for i := range schemas {
		for name, table := range schemas[i].Tables {
			if _, found := tables[name]; found {
				return nil, fmt.Errorf("%w:table %q already there", ErrMergeSchema, name)
			}
			tables[name] = table
		}
	}

	type mapRelations = map[dataType][]Relation

	mergeRelationFunc := func(getRelationsFunc func(*DBSchema) mapRelations, schemas ...*DBSchema) map[dataType][]Relation {
		allRels := map[dataType][]Relation{}
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
	}

	err := result.Validate()
	if err != nil {
		return nil, fmt.Errorf("%w:%s", ErrMergeSchema, err.Error())
	}
	return &result, nil
}

func DropRelations(schema *DBSchema) *DBSchema {
	schema.MandatoryForeignKeys = nil
	schema.CascadeDeletes = nil
	schema.CheckingRelations = nil
	return schema
}
