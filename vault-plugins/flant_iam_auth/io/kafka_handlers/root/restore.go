package root

import (
	"encoding/json"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func HandleRestoreMessagesRootSource(txn *memdb.Txn, objType string, data []byte) error {
	var inputObject interface{}

	switch objType {
	case iam_model.UserType:
		inputObject = &iam_model.User{}
	case iam_model.ServiceAccountType:
		inputObject = &iam_model.ServiceAccount{}
	case iam_model.ProjectType:
		inputObject = &iam_model.Project{}
	case iam_model.TenantType:
		inputObject = &iam_model.Tenant{}
	case iam_model.FeatureFlagType:
		inputObject = &iam_model.FeatureFlag{}
	case iam_model.GroupType:
		inputObject = &iam_model.Group{}
	case iam_model.RoleType:
		inputObject = &iam_model.Role{}
	case iam_model.RoleBindingType:
		inputObject = &iam_model.RoleBinding{}
	case iam_model.RoleBindingApprovalType:
		inputObject = &iam_model.RoleBindingApproval{}
	case iam_model.MultipassType:
		inputObject = &iam_model.Multipass{}
	case iam_model.ServiceAccountPasswordType:
		inputObject = &iam_model.ServiceAccountPassword{}
	case iam_model.IdentitySharingType:
		inputObject = &iam_model.IdentitySharing{}
		// EXTENSION_SERVER_ACCESS
	case ext_model.ServerType:
		inputObject = &ext_model.Server{}
	default:
		return nil
	}
	table := objType
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
