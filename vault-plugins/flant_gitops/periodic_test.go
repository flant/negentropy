package flant_gitops

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	description string
	// precondition
	lastTimeStampInStorageShift time.Duration
	// run
	interval time.Duration
	// expected
	result bool
}

var testcases = []testCase{
	{
		description:                 "first run",
		lastTimeStampInStorageShift: -999999, // magic number
		interval:                    time.Second,
		result:                      true,
	},
	{
		description:                 "not exceed interval run",
		lastTimeStampInStorageShift: -time.Second,
		interval:                    2 * time.Second,
		result:                      false,
	},
	{
		description:                 "exceed interval run",
		lastTimeStampInStorageShift: -2 * time.Second,
		interval:                    1 * time.Second,
		result:                      true,
	},
}

func Test_checkExceedingInterval(t *testing.T) {
	ctx := context.Background()
	tb, err := getTestBackend(ctx)
	require.NoError(t, err)
	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			var savedTimeStamp time.Time
			if tc.lastTimeStampInStorageShift != -999999 {
				savedTimeStamp = tb.Clock.Now().Add(tc.lastTimeStampInStorageShift)
				err := setLastRunTimestamp(tb.Storage, savedTimeStamp)
				require.NoError(t, err)
			}

			gotResult, err := checkExceedingInterval(ctx, tb.Storage, tc.interval)

			require.NoError(t, err)
			require.Equal(t, tc.result, gotResult)
		})
	}
}

func setLastRunTimestamp(storage logical.Storage, now time.Time) error {
	ctx := context.TODO()
	err := storage.Put(ctx, &logical.StorageEntry{Key: lastPeriodicRunTimestampKey, Value: []byte(fmt.Sprintf("%d", now.Unix()))})
	return err
}
