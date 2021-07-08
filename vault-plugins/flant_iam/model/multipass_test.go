package model

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func Test_MultipassDbSchema(t *testing.T) {
	schema := MultipassSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("multipass schema is invalid: %v", err)
	}
}

func Test_MultipassMarshalling(t *testing.T) {
	flipflopMultipass := func(t *testing.T, token *Multipass, includeSensitive bool) *Multipass {
		j, err := json.Marshal(token)
		if err != nil {
			t.Fatalf("cannot marshal multipass without sensitive data: %v", err)
		}

		restored := &Multipass{}
		err = json.Unmarshal(j, restored)
		if err != nil {
			t.Fatalf("cannot unmarshal multipass back: %v", err)
		}

		return restored
	}

	initialMultipass := &Multipass{
		UUID:        uuid.New(),
		TenantUUID:  uuid.New(),
		OwnerUUID:   uuid.New(),
		OwnerType:   MultipassOwnerServiceAccount,
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
