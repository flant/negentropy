package root

import (
	"encoding/json"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type ModelHandler interface {
	HandleUser(user *iam_model.User) error
	HandleDeleteUser(uuid string) error

	HandleMultipass(mp *iam_model.Multipass) error
	HandleDeleteMultipass(uuid string) error

	HandleServiceAccount(sa *iam_model.ServiceAccount) error
	HandleDeleteServiceAccount(uuid string) error
}

func HandleNewMessageIamRootSource(txn *io.MemoryStoreTxn, handler ModelHandler, msg *kafka.MsgDecoded) error {
	isDelete := msg.IsDeleted()

	var inputObject interface{}
	var table string
	var entityHandler func() error

	objID := msg.ID

	switch msg.Type {
	case iam_model.UserType:
		table = iam_model.UserType
		user := &iam_model.User{}
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

	case iam_model.ServiceAccountType:
		table = iam_model.ServiceAccountType
		sa := &iam_model.ServiceAccount{}
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
	case iam_model.ProjectType:
		p := &iam_model.Project{}
		p.UUID = objID
		inputObject = p
		table = iam_model.ProjectType

	case iam_model.TenantType:
		t := &iam_model.Tenant{}
		t.UUID = objID
		inputObject = t
		table = iam_model.TenantType

	case iam_model.FeatureFlagType:
		t := &iam_model.FeatureFlag{}
		t.Name = objID
		inputObject = t
		table = iam_model.FeatureFlagType

	case iam_model.GroupType:
		t := &iam_model.Group{}
		t.UUID = objID
		inputObject = t
		table = iam_model.GroupType

	case iam_model.RoleType:
		t := &iam_model.Role{}
		t.Name = objID
		inputObject = t
		table = iam_model.RoleType

	case iam_model.RoleBindingType:
		t := &iam_model.RoleBinding{}
		t.UUID = objID
		inputObject = t
		table = iam_model.RoleBindingType

	case iam_model.RoleBindingApprovalType:
		t := &iam_model.RoleBindingApproval{}
		t.UUID = objID
		inputObject = t
		table = iam_model.RoleBindingApprovalType

	case iam_model.MultipassType:
		mp := &iam_model.Multipass{}
		mp.UUID = objID
		inputObject = mp
		table = iam_model.MultipassType
		if isDelete {
			entityHandler = func() error {
				return handler.HandleDeleteMultipass(objID)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleMultipass(mp)
			}
		}

	case iam_model.ServiceAccountPasswordType:
		t := &iam_model.ServiceAccountPassword{}
		t.UUID = objID
		inputObject = t
		table = iam_model.ServiceAccountPasswordType

	case iam_model.IdentitySharingType:
		t := &iam_model.IdentitySharing{}
		t.UUID = objID
		inputObject = t
		table = iam_model.IdentitySharingType

	case ext_model.ServerType:
		inputObject = &ext_model.Server{}
		table = ext_model.ServerType

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
