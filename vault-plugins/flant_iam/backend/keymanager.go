package backend

import (
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
)

type keyManager struct {
	// must never collide with any other field name
	idField   string
	entryName string
	parent    *keyManager
}

func (km *keyManager) IDField() string {
	return km.idField
}

func (km *keyManager) EntryName() string {
	return km.entryName
}

func (km *keyManager) GenerateID() string {
	return genUUID()
}

func (km *keyManager) EntryPattern() string {
	p := km.entryName + OptionalParamRegex(km.IDField())
	if km.parent != nil {
		return km.parent.prefixPattern() + "/" + p
	}
	return p
}

func (km *keyManager) prefixPattern() string {
	p := km.entryName + "/" + framework.GenericNameRegex(km.IDField())
	if km.parent != nil {
		return km.parent.prefixPattern() + "/" + p
	}
	return p
}

func (km *keyManager) ListPattern() string {
	p := km.entryName + "/?"
	if km.parent != nil {
		return km.parent.prefixPattern() + "/" + p
	}
	return p
}

// OptionalParamRegex should be just as strict as framework.GenericNameRegex, but optional
func OptionalParamRegex(name string) string {
	return fmt.Sprintf("(/%s)?", framework.GenericNameRegex(name))
}
