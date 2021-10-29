package iam_clients

import (
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type UserClient interface {
	Create(teammate model.Teammate) (*model.Teammate, error)
	Update(teammate model.Teammate) (*model.Teammate, error)
	Delete(uuid iam_model.UserUUID) (bool, error)
}

func NewUserClient() (UserClient, error) {
	return &userClient{}, nil
}

type userClient struct { // TODO IMPLEMENT IAM client
}

func (u userClient) Create(teammate model.Teammate) (*model.Teammate, error) {
	return &teammate, nil
}

func (u userClient) Update(teammate model.Teammate) (*model.Teammate, error) {
	return &teammate, nil
}

func (u userClient) Delete(uuid iam_model.UserUUID) (bool, error) {
	return true, nil
}
