package flant_gitops

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/vault/sdk/logical"

	"testing"
	"time"

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

func Test_checkAndUpdateTimeStamp(t *testing.T) {
	ctx := context.Background()
	_, storage, _, mockClock := getTestBackend(t, ctx)
	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			var savedTimeStamp time.Time
			if tc.lastTimeStampInStorageShift != -999999 {
				savedTimeStamp = mockClock.Now().Add(tc.lastTimeStampInStorageShift)
				err := setLastRunTimestamp(storage, savedTimeStamp)
				require.NoError(t, err)
			}

			gotResult, err := checkExceedingInterval(ctx, storage, tc.interval)

			require.NoError(t, err)
			require.Equal(t, tc.result, gotResult)
		})
	}
}

func getAndParseLastRunTimestamp(storage logical.Storage) (int64, error) {
	ctx := context.TODO()
	entry, err := storage.Get(ctx, lastPeriodicRunTimestampKey)
	if err != nil {
		return 0, fmt.Errorf("unable to get key %q from storage: %w", lastPeriodicRunTimestampKey, err)
	}
	if entry == nil {
		return 0, nil
	}
	return strconv.ParseInt(string(entry.Value), 10, 64)
}

func setLastRunTimestamp(storage logical.Storage, now time.Time) error {
	ctx := context.TODO()
	err := storage.Put(ctx, &logical.StorageEntry{Key: lastPeriodicRunTimestampKey, Value: []byte(fmt.Sprintf("%d", now.Unix()))})
	return err
}
