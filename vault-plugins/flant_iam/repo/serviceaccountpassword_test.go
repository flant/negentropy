package repo

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func Test_ServiceAccountPasswordMarshalling(t *testing.T) {
	flipflopSAP := func(t *testing.T, token *model.ServiceAccountPassword, includeSensitive bool) *model.ServiceAccountPassword {
		var res interface{}

		res = token
		if !includeSensitive {
			res = OmitSensitive(token)
		}
		j, err := json.Marshal(res)
		if err != nil {
			t.Fatalf("cannot marshal multipass without sensitive data: %v", err)
		}

		restored := &model.ServiceAccountPassword{}
		err = json.Unmarshal(j, restored)
		if err != nil {
			t.Fatalf("cannot unmarshal multipass back: %v", err)
		}

		return restored
	}

	initialSAP := &model.ServiceAccountPassword{
		UUID:        uuid.New(),
		TenantUUID:  uuid.New(),
		OwnerUUID:   uuid.New(),
		Description: "xxx",
		CIDRs:       []string{"10.0.0.0/24"},
		Roles:       []string{"root"},
		Secret:      "Pepper",
	}

	{
		restored := flipflopSAP(t, initialSAP, true)

		if !reflect.DeepEqual(initialSAP, restored) {
			t.Fatalf("expected to have the same multipass with sensitive data: was=%v, became=%v", initialSAP, restored)
		}
	}

	{
		restored := flipflopSAP(t, initialSAP, false)

		initialSAP.Secret = "" // clean what is expected to be sensitive
		if !reflect.DeepEqual(initialSAP, restored) {
			t.Fatalf("expected omitted sensitive data: was=%v, became=%v", initialSAP, restored)
		}
	}
}
