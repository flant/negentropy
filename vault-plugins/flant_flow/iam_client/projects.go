package iam_client

import (
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
)

type Projects interface {
	Create(client model.Project) (*model.Project, error)
	Update(client model.Project) (*model.Project, error)
	Delete(uuid model.ProjectUUID) (bool, error)
}

func NewProjectClient() (Projects, error) {
	return &projectClient{}, nil
}

type projectClient struct { // TODO IMPLEMENT IAM client
}

func (u projectClient) Create(project model.Project) (*model.Project, error) {
	return &project, nil
}

func (u projectClient) Update(project model.Project) (*model.Project, error) {
	return &project, nil
}

func (u projectClient) Delete(uuid model.ProjectUUID) (bool, error) {
	return true, nil
}
