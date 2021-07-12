package model

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/shared/io"
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
	Origin ObjectOrigin `json:""`

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

type ExtensionRepository struct {
	db *io.MemoryStoreTxn
}

func NewExtensionRepository(db *io.MemoryStoreTxn) *ExtensionRepository {
	return &ExtensionRepository{db: db}
}

func (r *ExtensionRepository) Create(ext *Extension) error {
	switch ext.OwnerType {
	case UserType:
		return NewUserRepository(r.db).SetExtension(ext)

	case ServiceAccountType:
		return NewServiceAccountRepository(r.db).SetExtension(ext)

	case RoleBindingType:
		return NewRoleBindingRepository(r.db).SetExtension(ext)

	case GroupType:
		return NewGroupRepository(r.db).SetExtension(ext)

	case MultipassType:
		return NewMultipassRepository(r.db).SetExtension(ext)
	}
	return fmt.Errorf("extension is not supported for type %q", ext.OwnerType)
}

func (r *ExtensionRepository) Delete(origin ObjectOrigin, ownerUUID OwnerUUID) error {
	repos := []extensionUnsetter{
		NewUserRepository(r.db),
		NewServiceAccountRepository(r.db),
		NewRoleBindingRepository(r.db),
		NewGroupRepository(r.db),
		NewMultipassRepository(r.db),
	}

	for _, repo := range repos {
		err := repo.UnsetExtension(origin, ownerUUID)
		if err == ErrNotFound {
			continue
		}
		return err
	}

	return fmt.Errorf("extension not found among supported types")
}

type extensionUnsetter interface {
	UnsetExtension(ObjectOrigin, OwnerUUID) error
}
