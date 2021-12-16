package io

type VaultApiAction struct {
	op func() error
}

func NewVaultApiAction(op func() error) *VaultApiAction {
	return &VaultApiAction{op: op}
}

func (a *VaultApiAction) Execute() error {
	return a.op()
}
