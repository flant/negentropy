package iam_clients

import (
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
)

type TenantClient interface {
	Create(client model.Client) (*model.Client, error)
	Update(client model.Client) (*model.Client, error)
	Delete(uuid model.ClientUUID) (bool, error)
}

func NewTenantClient() (TenantClient, error) {
	return &tenantClient{}, nil
}

type tenantClient struct { // TODO IMPLEMENT IAM client
}

func (u tenantClient) Create(client model.Client) (*model.Client, error) {
	return &client, nil
}

func (u tenantClient) Update(client model.Client) (*model.Client, error) {
	return &client, nil
}

func (u tenantClient) Delete(uuid model.ClientUUID) (bool, error) {
	return true, nil
}
