package flant_gitops

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

func getTestBackend(t *testing.T, ctx context.Context) (*backend, logical.Storage, *util.TestLogger) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	logical.TestBackendConfig()

	testLogger := util.NewTestLogger()

	config := &logical.BackendConfig{
		Logger: testLogger.VaultLogger,
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}

	b, err := newBackend(config)
	if err != nil {
		t.Fatalf("unable to create backend: %s", err)
	}

	if err := b.SetupBackend(ctx, config); err != nil {
		t.Fatalf("unable to setup backend: %s", err)
	}

	return b, config.StorageView, testLogger
}

func ListTasks(t *testing.T, ctx context.Context, b *backend, storage logical.Storage) []string {
	req := &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       "task",
		Storage:    storage,
		Connection: &logical.Connection{},
	}

	resp, err := b.HandleRequest(ctx, req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v\n", err, resp)
	}

	return resp.Data["keys"].([]string)
}

func GetTaskStatus(t *testing.T, ctx context.Context, b *backend, storage logical.Storage, uuid string) (string, string) {
	req := &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       fmt.Sprintf("task/%s", uuid),
		Storage:    storage,
		Connection: &logical.Connection{},
	}

	resp, err := b.HandleRequest(ctx, req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v\n", err, resp)
	}

	return resp.Data["status"].(string), resp.Data["reason"].(string)
}

func GetTaskLog(t *testing.T, ctx context.Context, b *backend, storage logical.Storage, uuid string) string {
	req := &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       fmt.Sprintf("task/%s/log", uuid),
		Storage:    storage,
		Connection: &logical.Connection{},
		Data: map[string]interface{}{
			"limit": 1000_000_000,
		},
	}

	resp, err := b.HandleRequest(ctx, req)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("err:%v resp:%#v\n", err, resp)
	}

	return resp.Data["result"].(string)
}

func WaitForTaskCompletion(t *testing.T, ctx context.Context, b *backend, storage logical.Storage, uuid string) {
	for {
		status, reason := GetTaskStatus(t, ctx, b, storage, uuid)

		fmt.Printf("Poll task %s: status=%s reason=%q\n", uuid, status, reason)

		switch status {
		case "QUEUED", "RUNNING":

		case "COMPLETED", "FAILED":
			return

		default:
			taskLog := GetTaskLog(t, ctx, b, storage, uuid)
			t.Fatalf("got unexpected task %s status %s reason %s:\n%s\n", uuid, status, reason, taskLog)
		}

		time.Sleep(1 * time.Second)
	}
}

func WaitForTaskSuccess(t *testing.T, ctx context.Context, b *backend, storage logical.Storage, uuid string) {
	for {
		status, reason := GetTaskStatus(t, ctx, b, storage, uuid)

		fmt.Printf("Poll task %s: status=%s reason=%q\n", uuid, status, reason)

		switch status {
		case "QUEUED", "RUNNING":

		case "SUCCEEDED":
			return

		default:
			taskLog := GetTaskLog(t, ctx, b, storage, uuid)
			t.Fatalf("got unexpected task %s status %s reason %s:\n%s\n", uuid, status, reason, taskLog)
		}

		time.Sleep(1 * time.Second)
	}
}

func WaitForTaskFailure(t *testing.T, ctx context.Context, b *backend, storage logical.Storage, uuid string) string {
	for {
		status, reason := GetTaskStatus(t, ctx, b, storage, uuid)

		fmt.Printf("Poll task %s: status=%s reason=%q\n", uuid, status, reason)

		switch status {
		case "QUEUED", "RUNNING":

		case "FAILED":
			return reason

		default:
			taskLog := GetTaskLog(t, ctx, b, storage, uuid)
			t.Fatalf("got unexpected task %s status %s reason %s:\n%s\n", uuid, status, reason, taskLog)
		}

		time.Sleep(1 * time.Second)
	}
}
