package multipass

import (
	"time"

	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type Payload struct {
	Description string        `json:"description"`
	TTL         time.Duration `json:"ttl"`
	MaxTTL      time.Duration `json:"max_ttl"`
}

func GetPayload() Payload {
	return Payload{
		Description: "desc - " + uuid.New(),
		TTL:         100 * time.Second,
		MaxTTL:      1000 * time.Second,
	}
}
