package self

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
)

func HandleRestoreMessagesSelfSource(txn *memdb.Txn, objType string, data []byte) error {
	var inputObject interface{}
	var table string

	// only write to mem storage
	switch objType {
	case model.AuthSourceType:
		table = model.AuthSourceType
		source := &model.AuthSource{}
		inputObject = source
	case model.AuthMethodType:
		table = model.AuthSourceType
		inputObject = &model.AuthMethod{}
	case model.EntityType:
		inputObject = &model.Entity{}
		table = model.EntityType
	case model.EntityAliasType:
		inputObject = &model.EntityAlias{}
		table = model.EntityAliasType
	case model.JWTIssueTypeType:
		inputObject = &model.JWTIssueType{}
		table = model.JWTIssueTypeType

	default:
		return nil
	}

	err := json.Unmarshal(data, inputObject)
	if err != nil {
		return err
	}

	err = txn.Insert(table, inputObject)
	if err != nil {
		return err
	}

	return nil
}