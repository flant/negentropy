package rolebinding

import "github.com/flant/negentropy/vault-plugins/flant_iam/uuid"

type Payload struct {
	Subjects   []subject `json:"subjects"`
	TTL        int       `json:"ttl"`
	RequireMFA bool      `json:"require_mfa"`
}

type subject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func GetPayload() Payload {
	return Payload{Subjects: []subject{{Type: "test", ID: uuid.New()}}, RequireMFA: false, TTL: 30}
}
