package rolebindingapproval

import "github.com/flant/negentropy/vault-plugins/flant_iam/uuid"

type Payload struct {
	Name interface{} `json:"name,omitempty"`
}

func GetPayload() Payload {
	return Payload{Name: uuid.New()}
}
