package flant_gitops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/git_repository"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

func invokePeriodicRun(t *testing.T, ctx context.Context, b *backend, testLogger *util.TestLogger, storage logical.Storage) {
	testLogger.Reset()

	//req := &logical.Request{
	//	Operation:  logical.ReadOperation,
	//	Path:       "",
	//	Data:       make(map[string]interface{}),
	//	Storage:    storage,
	//	Connection: &logical.Connection{},
	//}

	if err := b.PeriodicTask(storage); err != nil {
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

func getLastSuccessfulCommit(t *testing.T, ctx context.Context, storage logical.Storage) string {
	//entry, err := storage.Get(ctx, storageKeyLastSuccessfulCommit)
	//if err != nil {
	//	t.Fatalf("error getting key %q from storage: %s", lastPeriodicRunTimestampKey, err)
	//}
	//
	//if entry == nil {
	//	return ""
	//}
	//
	//return string(entry.Value)
	return ""
}

func TestPeriodic_PollOperation(t *testing.T) {
	ctx := context.Background()

	b, storage, testLogger, systemClockMock := getTestBackend(t, ctx)

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "configure",
		Data: map[string]interface{}{
			git_repository.FieldNameGitRepoUrl:                                 "no-such-repo",
			git_repository.FieldNameGitBranch:                                  "main",
			git_repository.FieldNameGitPollPeriod:                              "5m",
			git_repository.FieldNameRequiredNumberOfVerifiedSignaturesOnCommit: 0,
			git_repository.FieldNameInitialLastSuccessfulCommit:                "",
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
		//periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)
		//WaitForTaskCompletion(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

		if periodicTaskUUIDs[len(periodicTaskUUIDs)-1] == "" {
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

		//periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)

		WaitForTaskCompletion(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

		lastPeriodicRunTimestamp := getRequiredLastPeriodicRunTime(t, ctx, storage)

		if lastPeriodicRunTimestamp != lastPeriodicRunTimestampBeforeInvokation {
			t.Fatalf("periodic run timestamp should not change on second run before poll period: %d != %d", lastPeriodicRunTimestamp, lastPeriodicRunTimestampBeforeInvokation)
		}

		if periodicTaskUUIDs[1] != periodicTaskUUIDs[len(periodicTaskUUIDs)-1] {
			t.Fatalf("new periodic task should not be added after second periodic func invocation before git poll period")
		}
	}

	{
		// 5 minutes passed
		systemClockMock.NowTime = systemClockMock.NowTime.Add(5 * time.Minute)

		invokePeriodicRun(t, ctx, b, testLogger, storage)
		//periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)
		WaitForTaskCompletion(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

		lastPeriodicRunTimestamp := getRequiredLastPeriodicRunTime(t, ctx, storage)

		if lastPeriodicRunTimestamp != systemClockMock.Now().Unix() {
			t.Fatalf("unexpected last periodic run timestamp %d", lastPeriodicRunTimestamp)
		}

		if periodicTaskUUIDs[2] == periodicTaskUUIDs[1] {
			t.Fatalf("new periodic task should be added after third periodic func invocation after git poll period")
		}
	}
}

func TestPeriodic_DockerCommand(t *testing.T) {
	var periodicTaskUUIDs []string

	ctx := context.Background()

	b, storage, testLogger, systemClockMock := getTestBackend(t, ctx)

	testGitRepoDir := util.GenerateTmpGitRepo(t, "flant_gitops_test_repo")
	defer os.RemoveAll(testGitRepoDir)

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "configure",
		Data: map[string]interface{}{
			git_repository.FieldNameGitRepoUrl:                                 testGitRepoDir,
			git_repository.FieldNameGitBranch:                                  "main",
			git_repository.FieldNameGitPollPeriod:                              "5m",
			git_repository.FieldNameRequiredNumberOfVerifiedSignaturesOnCommit: 0,
			git_repository.FieldNameInitialLastSuccessfulCommit:                "",
		},
		Storage:    storage,
		Connection: &logical.Connection{},
	}

	resp, err := b.HandleRequest(ctx, req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v\n", err, resp)
	}

	util.ExecGitCommand(t, testGitRepoDir, "checkout", "-b", "main")
	util.WriteFileIntoDir(t, filepath.Join(testGitRepoDir, "data"), []byte("OUTPUT1\n"))
	util.ExecGitCommand(t, testGitRepoDir, "add", ".")
	util.ExecGitCommand(t, testGitRepoDir, "commit", "-m", "one")
	currentCommit := strings.TrimSpace(util.ExecGitCommand(t, testGitRepoDir, "rev-parse", "HEAD"))
	rememberFirstCommit := currentCommit

	fmt.Printf("Current commit in test repo %s: %s\n", testGitRepoDir, currentCommit)

	{
		invokePeriodicRun(t, ctx, b, testLogger, storage)
		//periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)
		WaitForTaskSuccess(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

		if match, _ := testLogger.Grep("OUTPUT1"); !match {
			t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
		}

		lastSuccessfulCommit := getLastSuccessfulCommit(t, ctx, storage)
		if lastSuccessfulCommit != currentCommit {
			t.Fatalf("expected last successful commit to equal %s, got %s", currentCommit, lastSuccessfulCommit)
		}
	}

	{
		testLogger.Reset()

		// 5 minutes passed
		systemClockMock.NowTime = systemClockMock.NowTime.Add(5 * time.Minute)

		invokePeriodicRun(t, ctx, b, testLogger, storage)
		//periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)
		WaitForTaskSuccess(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

		if match, _ := testLogger.Grep("Head commit not changed: skipping"); !match {
			t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
		}

		lastSuccessfulCommit := getLastSuccessfulCommit(t, ctx, storage)
		if lastSuccessfulCommit != currentCommit {
			t.Fatalf("expected last successful commit to equal %s, got %s", currentCommit, lastSuccessfulCommit)
		}
	}

	util.WriteFileIntoDir(t, filepath.Join(testGitRepoDir, "data"), []byte("OUTPUT2\n"))
	util.ExecGitCommand(t, testGitRepoDir, "add", ".")
	util.ExecGitCommand(t, testGitRepoDir, "commit", "-m", "two")
	currentCommit = strings.TrimSpace(util.ExecGitCommand(t, testGitRepoDir, "rev-parse", "HEAD"))

	{
		testLogger.Reset()

		// 5 minutes passed
		systemClockMock.NowTime = systemClockMock.NowTime.Add(5 * time.Minute)

		invokePeriodicRun(t, ctx, b, testLogger, storage)
		//periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)
		WaitForTaskSuccess(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

		if match, _ := testLogger.Grep("OUTPUT2"); !match {
			t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
		}

		lastSuccessfulCommit := getLastSuccessfulCommit(t, ctx, storage)
		if lastSuccessfulCommit != currentCommit {
			t.Fatalf("expected last successful commit to equal %s, got %s", currentCommit, lastSuccessfulCommit)
		}
	}

	{
		testLogger.Reset()

		// 5 minutes passed
		systemClockMock.NowTime = systemClockMock.NowTime.Add(5 * time.Minute)

		invokePeriodicRun(t, ctx, b, testLogger, storage)
		//periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)
		WaitForTaskSuccess(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

		if match, _ := testLogger.Grep("Head commit not changed: skipping"); !match {
			t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
		}

		lastSuccessfulCommit := getLastSuccessfulCommit(t, ctx, storage)
		if lastSuccessfulCommit != currentCommit {
			t.Fatalf("expected last successful commit to equal %s, got %s", currentCommit, lastSuccessfulCommit)
		}
	}

	util.ExecGitCommand(t, testGitRepoDir, "checkout", "-b", "saved_branch")
	util.ExecGitCommand(t, testGitRepoDir, "checkout", "main")
	util.ExecGitCommand(t, testGitRepoDir, "reset", "--hard", rememberFirstCommit)
	prevLastSuccessfulCommit := getLastSuccessfulCommit(t, ctx, storage)
	currentCommit = strings.TrimSpace(util.ExecGitCommand(t, testGitRepoDir, "rev-parse", "HEAD"))

	fmt.Printf("-- prevLastSuccessfulCommit=%s currentCommit=%s\n", prevLastSuccessfulCommit, currentCommit)

	{
		testLogger.Reset()

		// 5 minutes passed
		systemClockMock.NowTime = systemClockMock.NowTime.Add(5 * time.Minute)

		invokePeriodicRun(t, ctx, b, testLogger, storage)
		//periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)
		reason := WaitForTaskFailure(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

		expectedReason := fmt.Sprintf("unable to run periodic task for git commit %q which is not descendant of the last successfully processed commit %q", currentCommit, prevLastSuccessfulCommit)
		if !strings.Contains(reason, expectedReason) {
			t.Fatalf("got unexpected failure reason:\n%q\nexpected:\n%q\n", reason, expectedReason)
		}

		lastSuccessfulCommit := getLastSuccessfulCommit(t, ctx, storage)
		if lastSuccessfulCommit != prevLastSuccessfulCommit {
			t.Fatalf("expected last successful commit to equal %s, got %s, current repo commit is %s", prevLastSuccessfulCommit, lastSuccessfulCommit, currentCommit)
		}
	}
}
