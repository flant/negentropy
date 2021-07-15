package service_account

import (
	"time"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

type Payload struct {
	Version     string        `json:"resource_version"`
	Identifier  string        `json:"identifier"`
	CIDRs       []string      `json:"allowed_cidrs"`
	TokenTTL    time.Duration `json:"token_ttl"`
	TokenMaxTTL time.Duration `json:"token_max_ttl"`
}

func GetPayload() Payload {
	return Payload{
		Version:     tools.RandomStr(),
		Identifier:  uuid.New(),
		CIDRs:       []string{"0.0.0.0/0"},
		TokenTTL:    100 * time.Second,
		TokenMaxTTL: 1000 * time.Second,
	}
}
