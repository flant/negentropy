package key

import (
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
)

type Manager struct {
	// must never collide with any other field name
	idField   string
	entryName string
	parent    *Manager
}

func NewManager(idField, entryName string) *Manager {
	return &Manager{
		idField:   idField,
		entryName: entryName,
	}
}

func (km *Manager) Child(idField, entryName string) *Manager {
	return &Manager{
		idField:   idField,
		entryName: entryName,
		parent:    km,
	}
}

func (km *Manager) IDField() string {
	return km.idField
}

func (km *Manager) EntryName() string {
	return km.entryName
}

func (km *Manager) EntryPattern() string {
	p := km.entryName + OptionalParamRegex(km.IDField())
	if km.parent != nil {
		return km.parent.prefixPattern() + "/" + p
	}
	return p
}

func (km *Manager) prefixPattern() string {
	p := km.entryName + "/" + framework.GenericNameRegex(km.IDField())
	if km.parent != nil {
		return km.parent.prefixPattern() + "/" + p
	}
	return p
}

func (km *Manager) ListPattern() string {
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
