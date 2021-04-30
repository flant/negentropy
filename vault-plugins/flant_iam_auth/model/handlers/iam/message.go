package iam

import (
	"encoding/json"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ModelHandler interface {
	HandleUser(user *iam.User) error
	HandleServiceAccount(sa *iam.ServiceAccount) error
}

func HandleNewMessageIamRootSource(txn *io.MemoryStoreTxn, handler ModelHandler, objType string, objID string, data []byte) error {
	var inputObject interface{}
	var table string
	var entityHandler func() error

	switch objType {
	case iam.UserType:
		table = iam.UserType
		user := &iam.User{}
		user.UUID = objID
		inputObject = user
		// dont call here because we need unmarshal and add object in mem storage before handler
		entityHandler = func() error {
			return handler.HandleUser(user)
		}
	case iam.ServiceAccountType:
		table = iam.ServiceAccountType
		sa := &iam.ServiceAccount{}
		sa.UUID = objID
		inputObject = sa
		entityHandler = func() error {
			return handler.HandleServiceAccount(sa)
		}
	case iam.ProjectType:
		inputObject = &iam.Project{}
		table = iam.ProjectType
	case iam.TenantType:
		inputObject = &iam.Tenant{}
		table = iam.TenantType

	default:
		return nil
	}

	if len(data) == 0 {
		// todo delete object
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

	if entityHandler != nil {
		return entityHandler()
	}

	return nil
}
