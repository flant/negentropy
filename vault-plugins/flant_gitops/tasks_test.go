package flant_gitops

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

// TODO: Add methods into tasks queue interface instead

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
