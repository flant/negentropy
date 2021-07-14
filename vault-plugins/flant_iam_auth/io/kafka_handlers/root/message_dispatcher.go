package root

import (
	"encoding/json"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type ModelHandler interface {
	HandleUser(user *iam.User) error
	HandleServiceAccount(sa *iam.ServiceAccount) error

	HandleDeleteUser(uuid string) error
	HandleDeleteServiceAccount(uuid string) error
}

func HandleNewMessageIamRootSource(txn *io.MemoryStoreTxn, handler ModelHandler, msg *kafka.MsgDecoded) error {
	isDelete := msg.IsDeleted()

	var inputObject interface{}
	var table string
	var entityHandler func() error

	objID := msg.ID

	switch msg.Type {
	case iam.UserType:
		table = iam.UserType
		user := &iam.User{}
		user.UUID = objID
		inputObject = user
		// dont call here because we need unmarshal and add object in mem storage before handler
		if isDelete {
			entityHandler = func() error {
				return handler.HandleDeleteUser(objID)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleUser(user)
			}
		}

	case iam.ServiceAccountType:
		table = iam.ServiceAccountType
		sa := &iam.ServiceAccount{}
		sa.UUID = objID
		inputObject = sa
		if isDelete {
			entityHandler = func() error {
				return handler.HandleDeleteServiceAccount(objID)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleServiceAccount(sa)
			}
		}
	case iam.ProjectType:
		p := &iam.Project{}
		p.UUID = objID
		inputObject = p
		table = iam.ProjectType
	case iam.TenantType:
		t := &iam.Tenant{}
		t.UUID = objID
		inputObject = t
		table = iam.TenantType

	default:
		return nil
	}

	if isDelete {
		err := txn.Delete(table, inputObject)
		if err != nil {
			return err
		}

		if entityHandler != nil {
			return entityHandler()
		}

		return nil
	}

	err := json.Unmarshal(msg.Data, inputObject)
	if err != nil {
		return err
	}

	err = txn.Insert(table, inputObject)
	if err != nil {
		return err
	}

	// TODO revert after debug
	//if entityHandler != nil {
	//	return entityHandler()
	//}

	return nil
}
