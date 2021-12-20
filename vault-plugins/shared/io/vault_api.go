package io

import (
	"time"

	"github.com/cenkalti/backoff"
)

func ThirtySecondsBackoff() backoff.BackOff {
	backoffRequest := backoff.NewExponentialBackOff()
	backoffRequest.MaxElapsedTime = time.Second * 30
	return backoffRequest
}

func FiveSecondsBackoff() backoff.BackOff {
	backoffRequest := backoff.NewExponentialBackOff()
	backoffRequest.MaxElapsedTime = time.Second * 5
	return backoffRequest
}

type VaultApiAction struct {
	op func() error
}

func NewVaultApiAction(op func() error) *VaultApiAction {
	return &VaultApiAction{op: op}
}

func (a *VaultApiAction) Execute() error {
	return a.op()
}
