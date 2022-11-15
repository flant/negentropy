package self

import (
	"encoding/json"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RestoreFunc func(io.Txn, io.MsgDecoded) (bool, error)

func HandleRestoreMessagesSelfSource(txn io.Txn, msg io.MsgDecoded, extraRestoreHandlers []RestoreFunc) error {
	var inputObject interface{}
	var table string

	for _, r := range extraRestoreHandlers {
		handled, err := r(txn, msg)
		if err != nil {
			return err
		}

		if handled {
			return nil
		}
	}

	// only write to mem storage
	switch msg.Type {
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
	case model.PolicyType:
		inputObject = &model.Policy{}
	default:
		return nil
	}
	table = msg.Type

	err := json.Unmarshal(msg.Data, inputObject)
	if err != nil {
		return err
	}

	err = txn.Insert(table, inputObject)
	if err != nil {
		return err
	}

	return nil
}
