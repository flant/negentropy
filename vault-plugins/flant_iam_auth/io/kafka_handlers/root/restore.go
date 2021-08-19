package root

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func HandleRestoreMessagesRootSource(txn *memdb.Txn, objType string, data []byte) error {
	var inputObject interface{}
	var table string

	switch objType {
	case iam_model.UserType:
		table = iam_model.UserType
		inputObject = &iam_model.User{}
	case iam_model.ServiceAccountType:
		table = iam_model.ServiceAccountType
		inputObject = &iam_model.ServiceAccount{}
	case iam_model.ProjectType:
		table = iam_model.ProjectType
		inputObject = &iam_model.Project{}
	case iam_model.TenantType:
		inputObject = &iam_model.Project{}
		table = iam_model.TenantType
	case iam_model.FeatureFlagType:
		inputObject = &iam_model.FeatureFlag{}
		table = iam_model.FeatureFlagType
	case iam_model.GroupType:
		inputObject = &iam_model.Group{}
		table = iam_model.GroupType
	case iam_model.RoleType:
		inputObject = &iam_model.Role{}
		table = iam_model.RoleType
	case iam_model.RoleBindingType:
		inputObject = &iam_model.RoleBinding{}
		table = iam_model.RoleBindingType
	case iam_model.RoleBindingApprovalType:
		inputObject = &iam_model.RoleBindingApproval{}
		table = iam_model.RoleBindingApprovalType
	case iam_model.MultipassType:
		inputObject = &iam_model.Multipass{}
		table = iam_model.MultipassType
	case iam_model.ServiceAccountPasswordType:
		inputObject = &iam_model.ServiceAccountPassword{}
		table = iam_model.ServiceAccountPasswordType
	case iam_model.IdentitySharingType:
		inputObject = &iam_model.IdentitySharing{}
		table = iam_model.IdentitySharingType
	case ext_model.ServerType:
		inputObject = &ext_model.Server{}
		table = ext_model.ServerType
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
