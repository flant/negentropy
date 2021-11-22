package repo

import (
	jwt "github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

const (
	// PK is the alias for "id. Index "id" is required by all tables.
	// In the domain, the primary key is not always "id".
	PK = "id"
)

func GetSchema() (*memdb.DBSchema, error) {
	jwtSchema, err := jwt.GetSchema(false)
	if err != nil {
		return nil, err
	}
	schema, err := memdb.MergeDBSchemas(
		TenantSchema(),
		ProjectSchema(),
		GroupSchema(),
		UserSchema(),
		FeatureFlagSchema(),
		ServiceAccountSchema(),
		RoleSchema(),
		RoleBindingSchema(),
		RoleBindingApprovalSchema(),
		MultipassSchema(),
		ServiceAccountPasswordSchema(),
		IdentitySharingSchema(),

		ReplicaSchema(),
		jwtSchema,
	)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func NewResourceVersion() string {
	return uuid.New()
}
