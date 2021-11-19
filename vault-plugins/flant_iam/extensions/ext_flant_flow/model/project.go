package model

import iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const ProjectType = "flow_project" // also, memdb schema name

type Project struct {
	iam_model.Project
	ServicePacks map[ServicePackName]string `json:"service_packs"`
}
