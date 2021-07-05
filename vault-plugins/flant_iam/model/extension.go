package model

import (
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

type ExtensionOwnerType string

const (
	ExtensionType       = "extension" // also, memdb schema name
	ExtensionOwnerIndex = "owner"

	ExtensionOwnerTypeUser                    ExtensionOwnerType = "user"
	ExtensionOwnerTypeServiceAccount          ExtensionOwnerType = "service_account"
	ExtensionOwnerTypeServiceAccountMultipass ExtensionOwnerType = "service_account_multipass"
	ExtensionOwnerTypeRoleBinding             ExtensionOwnerType = "role_binding"
	ExtensionOwnerTypeGroup                   ExtensionOwnerType = "group"
)

type Extension struct {
	UUID    string `json:"uuid"` // PK
	Version string `json:"resource_version"`

	// Origin is the source where the extension originates from
	Origin string `json:"origin"`

	// OwnerType is the object type to which the extension belongs to, e.g. "User" or "ServiceAccount".
	OwnerType ExtensionOwnerType `json:"owner_type"`
	// OwnerUUID is the id of an owner object
	OwnerUUID string `json:"owner_uuid"`

	// Attributes is the data to pass to other systems transparently
	Attributes map[string]interface{} `json:"attributes"`
	// SensitiveAttributes is the data to pass to some other systems transparently
	SensitiveAttributes map[string]interface{} `json:"sensitive_attributes"`
}

func (t *Extension) ObjType() string {
	return ExtensionType
}

func (t *Extension) ObjId() string {
	return t.UUID
}

func (t *Extension) Marshal(_ bool) ([]byte, error) {
	// TODO exclude sensitive data
	return jsonutil.EncodeJSON(t)
}

func (t *Extension) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, t)
	return err
}
