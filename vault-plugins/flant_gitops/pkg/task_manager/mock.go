package task_manager

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/client"
)

// this mock should work through requests into backend itself (not to vault)
type mockTaskManagerService struct {
	b       *framework.Backend
	storage logical.Storage
}

func (m *mockTaskManagerService) SaveTask(ctx context.Context, task taskUUID, commit hashCommit) error {
	return saveTask(ctx, m.storage, task, commit)
}

func (m *mockTaskManagerService) CheckTask(ctx context.Context, commit hashCommit) (taskExist, taskIsFinished, error) {
	return checkTask(ctx, m.storage, commit, m.readTaskStatus)
}

func NewMock(b *framework.Backend) func(logical.Storage, client.VaultClientController) TaskService {
	mock := &mockTaskManagerService{b: b}
	return func(storage logical.Storage, _ client.VaultClientController) TaskService {
		mock.storage = storage
		return mock
	}
}

func (m *mockTaskManagerService) readTaskStatus(task taskUUID) (string, error) {
	resp, err := m.b.HandleRequest(context.Background(), &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       "task/" + task,
		Storage:    m.storage,
		Connection: &logical.Connection{},
	})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Data == nil {
		return "", fmt.Errorf("emty response: %#v", resp)
	}
	return parse(resp.Data)
}
