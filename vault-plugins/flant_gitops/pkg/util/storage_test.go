package util

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

func Test_GetString(t *testing.T) {

	type testcase struct {
		description   string
		keyToSave     string
		valueToSave   string
		keyToRead     string
		expectedValue string
	}

	var tests = []testcase{
		{
			description:   "normal read",
			keyToSave:     "k1",
			valueToSave:   "v1",
			keyToRead:     "k1",
			expectedValue: "v1",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ctx := context.Background()
			storage := &logical.InmemStorage{}
			err := storage.Put(ctx, &logical.StorageEntry{
				Key:   test.keyToSave,
				Value: []byte(test.valueToSave),
			})
			require.NoError(t, err)

			got, err := GetString(ctx, storage, test.keyToRead)
			require.NoError(t, err)

			require.Equal(t, got, test.expectedValue)
		})
	}
}

func Test_PutString(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	err := PutString(ctx, storage, "k1", "v1")
	require.NoError(t, err)

}
