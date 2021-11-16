package repo

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func Test_MultipassMarshalling(t *testing.T) {
	flipflopMultipass := func(t *testing.T, token *model.Multipass, includeSensitive bool) *model.Multipass {
		var res interface{}
		res = token
		if !includeSensitive {
			res = OmitSensitive(token)
		}
		j, err := json.Marshal(res)
		if err != nil {
			t.Fatalf("cannot marshal multipass without sensitive data: %v", err)
		}

		restored := &model.Multipass{}
		err = json.Unmarshal(j, restored)
		if err != nil {
			t.Fatalf("cannot unmarshal multipass back: %v", err)
		}

		return restored
	}

	initialMultipass := &model.Multipass{
		UUID:        uuid.New(),
		TenantUUID:  uuid.New(),
		OwnerUUID:   uuid.New(),
		OwnerType:   model.MultipassOwnerServiceAccount,
		Description: "xxx",
		TTL:         time.Hour,
		MaxTTL:      24 * time.Hour,
		CIDRs:       []string{"10.0.0.0/24"},
		Roles:       []string{"root"},
		Salt:        "Pepper",
	}

	{
		restored := flipflopMultipass(t, initialMultipass, true)

		if !reflect.DeepEqual(initialMultipass, restored) {
			t.Fatalf("expected to have the same multipass with sensitive data: was=%v, became=%v", initialMultipass, restored)
		}
	}

	{
		restored := flipflopMultipass(t, initialMultipass, false)

		initialMultipass.Salt = "" // clean what is expected to be sensitive
		if !reflect.DeepEqual(initialMultipass, restored) {
			t.Fatalf("expected omitted sensitive data: was=%v, became=%v", initialMultipass, restored)
		}
	}
}
