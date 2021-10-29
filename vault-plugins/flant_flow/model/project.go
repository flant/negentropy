package model

import iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const ProjectType = "project" // also, memdb schema name

type Project struct {
	iam_model.Project
	ServicePacks map[ServicePackName]string `json:"service_packs"`
}

func (p *Project) IsDeleted() bool {
	return p.ArchivingTimestamp != 0
}

func (p *Project) ObjType() string {
	return ProjectType
}

func (p *Project) ObjId() string {
	return p.UUID
}
