package model

import (
	"reflect"
	"testing"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func Test_TokenDbSchema(t *testing.T) {
	schema := UserSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("token schema is invalid: %v", err)
	}
}

func Test_TokenMarshalling(t *testing.T) {
	flipflopToken := func(t *testing.T, token *Token, includeSensitive bool) *Token {
		j, err := token.Marshal(includeSensitive)
		if err != nil {
			t.Fatalf("cannot marshal token without sensitive data: %v", err)
		}

		restored := &Token{}
		err = restored.Unmarshal(j)
		if err != nil {
			t.Fatalf("cannot unmarshal token back: %v", err)
		}

		return restored
	}

	initialToken := &Token{
		UUID:        uuid.New(),
		TenantUUID:  uuid.New(),
		OwnerUUID:   uuid.New(),
		OwnerType:   TokenOwnerServiceAccount,
		Description: "xxx",
		TTL:         time.Hour,
		MaxTTL:      24 * time.Hour,
		CIDRs:       []string{"10.0.0.0/24"},
		Roles:       []string{"root"},
		Salt:        "Pepper",
	}

	{
		restored := flipflopToken(t, initialToken, true)

		if !reflect.DeepEqual(initialToken, restored) {
			t.Fatalf("expected to have the same token with sensitive data: was=%v, became=%v", initialToken, restored)
		}
	}

	{
		restored := flipflopToken(t, initialToken, false)

		initialToken.Salt = "" // clean what is expected to be sensitive
		if !reflect.DeepEqual(initialToken, restored) {
			t.Fatalf("expected omitted sensitive data: was=%v, became=%v", initialToken, restored)
		}
	}
}
