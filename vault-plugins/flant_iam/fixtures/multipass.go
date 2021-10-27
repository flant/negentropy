package fixtures

import (
	"encoding/json"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func RandomUserMultipassCreatePayload() map[string]interface{} {
	multipass := model.Multipass{
		Description: "desc - " + uuid.New(),
		TTL:         100 * time.Second,
		MaxTTL:      1000 * time.Second,
		CIDRs:       nil,
		Roles:       nil,
	}

	bytes, _ := json.Marshal(multipass)
	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload) //nolint:errcheck
	return payload
}
