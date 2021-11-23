package model

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

type ExtensionOwnerType string

const (
	ExtensionType = "extension"

	ExtensionOwnerTypeUser           ExtensionOwnerType = UserType
	ExtensionOwnerTypeServiceAccount ExtensionOwnerType = ServiceAccountType
	ExtensionOwnerTypeRoleBinding    ExtensionOwnerType = RoleBindingType
	ExtensionOwnerTypeGroup          ExtensionOwnerType = GroupType
	ExtensionOwnerTypeMultipass      ExtensionOwnerType = MultipassType
)

func (eot ExtensionOwnerType) String() string {
	return string(eot)
}

type Extension struct {
	// Origin is the source where the extension originates from
	Origin consts.ObjectOrigin `json:""`

	// OwnerType is the object type to which the extension belongs to, e.g. "User" or "ServiceAccount".
	OwnerType ExtensionOwnerType `json:""`
	// OwnerUUID is the id of an owner object
	OwnerUUID OwnerUUID `json:""`

	// Attributes is the data to pass to other systems transparently
	Attributes map[string]interface{} `json:"attributes"`
	// SensitiveAttributes is the data to pass to some other systems transparently
	SensitiveAttributes map[string]interface{} `json:"sensitive_attributes,omitempty" sensitive:""`
}

func (e Extension) ObjType() string {
	return ExtensionType
}

func (e Extension) ObjId() string {
	return fmt.Sprintf("%s.%s.%s", e.Origin, e.OwnerType, e.OwnerUUID)
}
