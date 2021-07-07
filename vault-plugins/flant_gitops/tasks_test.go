package flant_gitops

import (
	"context"
	"fmt"
	"testing"

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

	return resp.Data["uuids"].([]string)
}

func GetTaskStatus(t *testing.T, ctx context.Context, b *backend, storage logical.Storage, uuid string) string {
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

	statusData := resp.Data["status"].(map[string]interface{})

	return statusData["Status"].(string)
}
