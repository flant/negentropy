package self

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
)

type RestoreFunc func(*memdb.Txn, string, []byte) (bool, error)

func HandleRestoreMessagesSelfSource(txn *memdb.Txn, objType string, data []byte, extraRestoreHandlers []RestoreFunc) error {
	var inputObject interface{}
	var table string

	for _, r := range extraRestoreHandlers {
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
		inputObject = &model.AuthSource{}
	case model.AuthMethodType:
		inputObject = &model.AuthMethod{}
	case model.EntityType:
		inputObject = &model.Entity{}
	case model.EntityAliasType:
		inputObject = &model.EntityAlias{}
	case model.JWTIssueTypeType:
		inputObject = &model.JWTIssueType{}
	default:
		return nil
	}
	table = objType

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
