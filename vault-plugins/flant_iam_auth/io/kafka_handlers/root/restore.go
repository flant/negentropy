package root

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
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
	case iam.FeatureFlagType:
		inputObject = &iam.FeatureFlag{}
		table = iam.FeatureFlagType
	case iam.GroupType:
		inputObject = &iam.Group{}
		table = iam.GroupType
	case iam.RoleType:
		inputObject = &iam.Role{}
		table = iam.RoleType
	case iam.RoleBindingType:
		inputObject = &iam.RoleBinding{}
		table = iam.RoleBindingType
	case iam.RoleBindingApprovalType:
		inputObject = &iam.RoleBindingApproval{}
		table = iam.RoleBindingApprovalType
	case iam.MultipassType:
		inputObject = &iam.Multipass{}
		table = iam.MultipassType
	case iam.ServiceAccountPasswordType:
		inputObject = &iam.ServiceAccountPassword{}
		table = iam.ServiceAccountPasswordType
	case iam.IdentitySharingType:
		inputObject = &iam.IdentitySharing{}
		table = iam.IdentitySharingType
	case model.ServerType:
		inputObject = &model.Server{}
		table = model.ServerType
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
