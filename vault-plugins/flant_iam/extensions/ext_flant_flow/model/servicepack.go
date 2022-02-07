package model

import (
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const ServicePackType = "servicepack" // also, memdb schema name

type ServicePack struct {
	memdb.ArchiveMark

	ProjectUUID      iam_model.ProjectUUID           `json:"uuid"`
	Name             ServicePackName                 `json:"service_pack_name"`
	Version          string                          `json:"resource_version"`
	Rolebindings     []iam_model.RoleBindingUUID     `json:"rolebindings"`
	IdentitySharings []iam_model.IdentitySharingUUID `json:"identity_sharings"`
}

func (u *ServicePack) ObjType() string {
	return ServicePackType
}

func (u *ServicePack) ObjId() string {
	return u.ProjectUUID + "_" + u.Name
}
