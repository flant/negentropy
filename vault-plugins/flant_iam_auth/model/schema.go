package model

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/hashicorp/go-memdb"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
)

const (
	ID       = "id" // required index by all tables
	ByName   = "name"
	ByUserID = "user_id"
)

func mergeSchema() (*memdb.DBSchema, error) {
	schema := EntitySchema()
	others := []*memdb.DBSchema{
		EntityAliasSchema(),
		AuthSourceSchema(),
		AuthMethodSchema(),
		JWTIssueTypeSchema(),
		MultipassGenerationNumberSchema(),

		iam.UserSchema(),
		iam.TenantSchema(),
		iam.ProjectSchema(),
		iam.ServiceAccountSchema(),
		iam.FeatureFlagSchema(),
		iam.GroupSchema(),
		iam.RoleSchema(),
		iam.RoleBindingSchema(),
		iam.RoleBindingApprovalSchema(),
		iam.MultipassSchema(),
		iam.ServiceAccountPasswordSchema(),
		iam.IdentitySharingSchema(),

		jwt.JWKSSchema(),
		model.ServerSchema(),
	}

	for _, o := range others {
		for name, table := range o.Tables {
			if _, ok := schema.Tables[name]; ok {
				return nil, fmt.Errorf("table %q already there", name)
			}
			schema.Tables[name] = table
		}
	}
	return schema, nil
}

func GetSchema() (*memdb.DBSchema, error) {
	return mergeSchema()
}

func NewResourceVersion() string {
	return uuid.New()
}
