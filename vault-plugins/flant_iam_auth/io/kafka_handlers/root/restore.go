package root

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func HandleRestoreMessagesRootSource(txn *memdb.Txn, objType string, data []byte) error {
	var inputObject interface{}
	var table string

	switch objType {
	case iam.UserType:
		table = iam.UserType
		inputObject = &iam.User{}
	case iam.ServiceAccountType:
		table = iam.ServiceAccountType
		inputObject = &iam.ServiceAccount{}
	case iam.ProjectType:
		table = iam.ProjectType
		inputObject = &iam.Project{}
	case iam.TenantType:
		inputObject = &iam.Project{}
		table = iam.TenantType

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
