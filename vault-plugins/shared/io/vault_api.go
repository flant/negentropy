package io

import (
	"github.com/hashicorp/vault/api"
)

type BackoffClientGetter func() (*api.Client, error)

type VaultApiAction struct {
	op func() error
}

func NewVaultApiAction(op func() error) *VaultApiAction {
	return &VaultApiAction{op: op}
}

func (a *VaultApiAction) Execute() error {
	return a.op()
}
