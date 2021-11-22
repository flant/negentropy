package repo

import (
	"fmt"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	// PK is the alias for "id. Index "id" is required by all tables.
	// In the domain, the primary key is not always "id".
	PK = "id"
)

func mergeTables() (map[string]*hcmemdb.TableSchema, error) {
	included := []map[string]*hcmemdb.TableSchema{
		// ClientSchema(),

		TeammateSchema(),
		// ProjectSchema(),
		ContactSchema(),
	}

	allTables := map[string]*hcmemdb.TableSchema{}

	for _, tables := range included {
		for name, table := range tables {
			if _, found := allTables[name]; found {
				return nil, fmt.Errorf("%w: table %q already there", memdb.ErrMergeSchema, name)
			}
			allTables[name] = table
		}
	}

	return allTables, nil
}

func GetSchema() (*memdb.DBSchema, error) {
	tables, err := mergeTables()
	if err != nil {
		return nil, err
	}
	schema := &memdb.DBSchema{
		Tables: tables,
		// TODO fill it
		MandatoryForeignKeys: nil,
		// TODO fill it
		CascadeDeletes: nil,
		// TODO fill it
		CheckingRelations: nil,
	}

	schema, err = memdb.MergeDBSchemas(
		schema,
		TeamSchema(),
	)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func NewResourceVersion() string {
	return uuid.New()
}
