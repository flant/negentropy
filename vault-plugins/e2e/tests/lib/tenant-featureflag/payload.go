package tenant_featureflag

import "github.com/flant/negentropy/vault-plugins/flant_iam/uuid"

type Payload struct {
	Identifier interface{} `json:"identifier,omitempty"`
}

func GetPayload() Payload {
	return Payload{Identifier: uuid.New()}
}
