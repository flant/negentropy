package model

import (
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func GetSchema() (*memdb.DBSchema, error) {
	iamSchema, err := model.GetSchema()
	if err != nil {
		return nil, err
	}

	iamSchema.Tables[ServerType] = ServerSchema().Tables[ServerType]

	return iamSchema, nil
}
