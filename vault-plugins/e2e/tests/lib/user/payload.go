package user

import (
	"fmt"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

type Payload struct {
	Identifier interface{} `json:"identifier,omitempty"`
	Email string		   `json:"email,omitempty"`
}

func GetPayload() Payload {
	return Payload{
		Identifier: uuid.New(),
		Email: fmt.Sprintf("%s@ex.com", tools.RandomStr()),
	}
}
