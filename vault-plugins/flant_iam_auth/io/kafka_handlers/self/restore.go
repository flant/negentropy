package self

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
)

type RestoreFunc func(*memdb.Txn, string, []byte) (bool, error)

func HandleRestoreMessagesSelfSource(txn *memdb.Txn, objType string, data []byte, restoreHandlers []RestoreFunc) error {
	var inputObject interface{}
	var table string

	for _, r := range restoreHandlers {
		handled, err := r(txn, objType, data)
		if err != nil {
			return err
		}

		if handled {
			return nil
		}
	}

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
