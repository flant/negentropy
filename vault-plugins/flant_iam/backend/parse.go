package backend

import (
	"github.com/hashicorp/vault/sdk/framework"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func getCreationID(expectID bool, data *framework.FieldData) string {
	var id string

	if expectID {
		// for privileged access
		id = data.Get("uuid").(string)
	}

	if id == "" {
		id = uuid.New()
	}

	return id
}
