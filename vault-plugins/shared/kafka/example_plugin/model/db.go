package model

import "github.com/hashicorp/go-memdb"

const (
	// PK is the alias for "id. Index "id" is required by all tables.
	// In the domain, the primary key is not always "id".
	PK = "id"
)

func GetSchema() (*memdb.DBSchema, error) {
	return UserSchema(), nil
}
