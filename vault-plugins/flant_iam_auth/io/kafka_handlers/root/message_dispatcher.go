package root

import (
	"encoding/json"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
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

	case iam.FeatureFlagType:
		t := &iam.FeatureFlag{}
		t.Name = objID
		inputObject = t
		table = iam.FeatureFlagType

	case iam.GroupType:
		t := &iam.Group{}
		t.UUID = objID
		inputObject = t
		table = iam.GroupType

	case iam.RoleType:
		t := &iam.Role{}
		t.Name = objID
		inputObject = t
		table = iam.RoleType

	case iam.RoleBindingType:
		t := &iam.RoleBinding{}
		t.UUID = objID
		inputObject = t
		table = iam.RoleBindingType

	case iam.RoleBindingApprovalType:
		t := &iam.RoleBindingApproval{}
		t.UUID = objID
		inputObject = t
		table = iam.RoleBindingApprovalType

	case iam.MultipassType:
		t := &iam.Multipass{}
		t.UUID = objID
		inputObject = t
		table = iam.MultipassType

	case iam.ServiceAccountPasswordType:
		t := &iam.ServiceAccountPassword{}
		t.UUID = objID
		inputObject = t
		table = iam.ServiceAccountPasswordType

	case iam.IdentitySharingType:
		t := &iam.IdentitySharing{}
		t.UUID = objID
		inputObject = t
		table = iam.IdentitySharingType

	case model.ServerType:
		inputObject = &model.Server{}
		table = model.ServerType

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

	if entityHandler != nil {
		return entityHandler()
	}

	return nil
}
