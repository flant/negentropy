package repo

import (
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	// PK is the alias for "id. Index "id" is required by all tables.
	// In the domain, the primary key is not always "id".
	PK = "id"
)

func GetSchema() (*memdb.DBSchema, error) {
	// schema has links to iam tables, so no opportunity to verify till merge with iam schema
	return memdb.MergeDBSchemas(false,
		TeamSchema(),
		TeammateSchema(),
		ContactSchema(),
		ServicePackSchema(),
	)
}

func NewResourceVersion() string {
	return uuid.New()
}
