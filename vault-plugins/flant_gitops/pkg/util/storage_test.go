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

	tests := []testcase{
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

func Test_StringMap(t *testing.T) {
	type testcase struct {
		description   string
		keyToSave     string
		valueToSave   map[string]string
		keyToRead     string
		expectedValue map[string]string
	}

	tests := []testcase{
		{
			description:   "normal read",
			keyToSave:     "k1",
			valueToSave:   map[string]string{"k": "v"},
			keyToRead:     "k1",
			expectedValue: map[string]string{"k": "v"},
		},
		{
			description:   "read written empty map",
			keyToSave:     "k1",
			valueToSave:   map[string]string{},
			keyToRead:     "k1",
			expectedValue: map[string]string{},
		},
		{
			description:   "read unwritten empty map",
			keyToSave:     "k1",
			valueToSave:   map[string]string{},
			keyToRead:     "k2",
			expectedValue: map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ctx := context.Background()
			storage := &logical.InmemStorage{}

			err := PutStringMap(ctx, storage, test.keyToSave, test.valueToSave)
			require.NoError(t, err)

			got, err := GetStringMap(ctx, storage, test.keyToRead)
			require.NoError(t, err)

			require.Equal(t, got, test.expectedValue)
		})
	}
}

func Test_ProhibitedWriteNilStringMap(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}

	err := PutStringMap(ctx, storage, "k1", nil)

	require.Error(t, err)
}
