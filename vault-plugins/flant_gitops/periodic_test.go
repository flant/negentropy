package flant_gitops

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

func invokePeriodicRun(t *testing.T, ctx context.Context, b *backend, testLogger *util.TestLogger, storage logical.Storage) {
	testLogger.Reset()

	runCtx, runCtxCancelFunc := context.WithCancel(ctx)
	defer runCtxCancelFunc()

	req := &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       "",
		Data:       make(map[string]interface{}),
		Storage:    storage,
		Connection: &logical.Connection{},
	}

	if err := b.PeriodicFunc(runCtx, req); err != nil {
		t.Fatalf("error running backend periodic task: %s", err)
	}
}

func getRequiredLastPeriodicRunTime(t *testing.T, ctx context.Context, storage logical.Storage) int64 {
	entry, err := storage.Get(ctx, lastPeriodicRunTimestampKey)
	if err != nil {
		t.Fatalf("error getting key %q from storage: %s", lastPeriodicRunTimestampKey, err)
	}

	if entry == nil {
		t.Fatalf("storage record by key %s not found", lastPeriodicRunTimestampKey)
	}

	timestamp, err := strconv.ParseInt(string(entry.Value), 10, 64)
	if err != nil {
		t.Fatalf("invalid record by key %s: expected timestamp int, got %q: %s", lastPeriodicRunTimestampKey, entry.Value, err)
	}

	return timestamp
}

func TestPeriodic_PollOperation(t *testing.T) {
	systemClockMock := util.NewFixedClock(time.Now())
	systemClock = systemClockMock

	ctx := context.Background()

	b, storage, testLogger := getTestBackend(t, ctx)

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "configure",
		Data: map[string]interface{}{
			fieldNameGitRepoUrl:    "/tmp/myrepo",
			fieldNameGitBranch:     "main",
			fieldNameGitPollPeriod: 5, // FIXME
			fieldNameRequiredNumberOfVerifiedSignaturesOnCommit: 0,
			fieldNameInitialLastSuccessfulCommit:                "",
			fieldNameDockerImage:                                "alpine:3.14.0@sha256:234cb88d3020898631af0ccbbcca9a66ae7306ecd30c9720690858c1b007d2a0",
			fieldNameCommand:                                    "echo DONE",
		},
		Storage:    storage,
		Connection: &logical.Connection{},
	}

	resp, err := b.HandleRequest(ctx, req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v\n", err, resp)
	}

	var periodicTaskUUIDs []string

	{
		entry, err := storage.Get(ctx, lastPeriodicRunTimestampKey)
		if err != nil {
			t.Fatalf("error getting key %q from storage: %s", lastPeriodicRunTimestampKey, err)
		}
		if entry != nil {
			t.Fatalf("found unexpected storage record by key %s: %s", lastPeriodicRunTimestampKey, entry.Value)
		}

		invokePeriodicRun(t, ctx, b, testLogger, storage)
		periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)

		if periodicTaskUUIDs[0] == "" {
			t.Fatalf("unexpected empty task uuid after first periodic run")
		}

		lastPeriodicRunTimestamp := getRequiredLastPeriodicRunTime(t, ctx, storage)

		if lastPeriodicRunTimestamp != systemClockMock.Now().Unix() {
			t.Fatalf("unexpected last periodic run timestamp %d", lastPeriodicRunTimestamp)
		}
	}

	{
		// 1 minute passed
		systemClockMock.NowTime = systemClockMock.NowTime.Add(1 * time.Minute)

		lastPeriodicRunTimestampBeforeInvokation := getRequiredLastPeriodicRunTime(t, ctx, storage)

		invokePeriodicRun(t, ctx, b, testLogger, storage)
		periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)

		lastPeriodicRunTimestamp := getRequiredLastPeriodicRunTime(t, ctx, storage)

		if lastPeriodicRunTimestamp != lastPeriodicRunTimestampBeforeInvokation {
			t.Fatalf("periodic run timestamp should not change on second run before poll period: %d != %d", lastPeriodicRunTimestamp, lastPeriodicRunTimestampBeforeInvokation)
		}

		if periodicTaskUUIDs[1] != periodicTaskUUIDs[0] {
			t.Fatalf("new periodic task should not be added after second periodic func invocation before git poll period")
		}

		if match, _ := testLogger.Grep("Added new periodic task with uuid"); match {
			t.Fatalf("new periodic task should not be added after second periodic func invocation")
		}
	}

	{
		// 5 minutes passed
		systemClockMock.NowTime = systemClockMock.NowTime.Add(5 * time.Minute)

		invokePeriodicRun(t, ctx, b, testLogger, storage)
		periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)

		lastPeriodicRunTimestamp := getRequiredLastPeriodicRunTime(t, ctx, storage)

		if lastPeriodicRunTimestamp != systemClockMock.Now().Unix() {
			t.Fatalf("unexpected last periodic run timestamp %d", lastPeriodicRunTimestamp)
		}

		if periodicTaskUUIDs[2] == periodicTaskUUIDs[1] {
			t.Fatalf("new periodic task should be added after third periodic func invocation after git poll period")
		}
	}
}
