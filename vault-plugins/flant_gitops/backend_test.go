package flant_gitops

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/kube"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/task_manager"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

type TestableBackend struct {
	B               *backend
	Storage         logical.Storage
	Logger          *util.TestLogger
	Clock           *util.MockClock
	MockKubeService *kube.MockKubeService
}

// getTestBackend prepare and returns test backend with mocked systemClock
func getTestBackend(ctx context.Context) (*TestableBackend, error) {
	mockedSystemClock, systemClockMock := util.NewMockedClock(time.Now())
	systemClock = mockedSystemClock // replace value of global variable for system time operating

	var kubeServiceMock *kube.MockKubeService
	kubeServiceProvider, kubeServiceMock = kube.NewMock()

	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	logical.TestBackendConfig()

	testLogger := util.NewTestLogger()

	storage := &logical.InmemStorage{}
	config := &logical.BackendConfig{
		Logger: testLogger.VaultLogger,
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: storage,
	}

	b, err := newBackend(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create backend: %w", err)
	}

	if err := b.SetupBackend(ctx, config); err != nil {
		return nil, fmt.Errorf("unable to setup backend: %s", err)
	}

	taskManagerServiceProvider = task_manager.NewMock(b.Backend)

	return &TestableBackend{
		B:               b,
		Storage:         storage,
		Logger:          testLogger,
		Clock:           systemClockMock,
		MockKubeService: kubeServiceMock,
	}, nil
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
