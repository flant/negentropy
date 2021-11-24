package model

import iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"

// Project is stored at memdb as a regular iam.Project with extension
type Project struct {
	iam_model.Project
	ServicePacks map[ServicePackName]string `json:"service_packs"`
}
