package task_manager

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/shared/client"
)

// store map[hashCommit]task_uuid
const storageKeyTasks = "commits_tasks"

type (
	taskUUID       = string
	hashCommit     = string
	taskExist      = bool
	taskIsFinished = bool
)

type TaskService interface {
	SaveTask(context.Context, taskUUID, hashCommit) error
	CheckTask(context.Context, hashCommit) (taskExist, taskIsFinished, error)
}

type service struct {
	storage                   logical.Storage
	accessVaultClientProvider client.VaultClientController
}

func Service(storage logical.Storage, accessVaultClientProvider client.VaultClientController) TaskService {
	return &service{
		storage:                   storage,
		accessVaultClientProvider: accessVaultClientProvider,
	}
}

func (s *service) SaveTask(ctx context.Context, task taskUUID, commit hashCommit) error {
	return saveTask(ctx, s.storage, task, commit)
}

func saveTask(ctx context.Context, storage logical.Storage, task taskUUID, commit hashCommit) error {
	tasks, err := util.GetStringMap(ctx, storage, storageKeyTasks)
	if err != nil {
		return fmt.Errorf("getting tasks from storage: %w", err)
	}
	if tasks == nil {
		tasks = map[hashCommit]taskUUID{}
	}
	tasks[commit] = task
	return util.PutStringMap(ctx, storage, storageKeyTasks, tasks)
}

func (s *service) CheckTask(ctx context.Context, commit hashCommit) (taskExist, taskIsFinished, error) {
	return checkTask(ctx, s.storage, commit, s.readTaskStatus)
}

func checkTask(ctx context.Context, storage logical.Storage, commit hashCommit,
	taskStatusProvider func(task taskUUID) (string, error)) (taskExist, taskIsFinished, error) {
	tasks, err := util.GetStringMap(ctx, storage, storageKeyTasks)
	if err != nil || tasks == nil {
		return false, false, fmt.Errorf("getting tasks from storage: %w", err)
	}
	task, ok := tasks[commit]
	if !ok {
		return false, false, nil
	}
	status, err := taskStatusProvider(task)
	if err != nil || tasks == nil {
		return true, false, fmt.Errorf("getting task %q status: %w", task, err)
	}
	finishedStatuses := map[string]struct{}{"SUCCEEDED": {}, "FAILED": {}, "CANCELED": {}}
	if _, ok := finishedStatuses[status]; ok {
		return true, true, nil
	}
	return true, false, nil
}

func (s *service) readTaskStatus(task taskUUID) (string, error) {
	cl, err := s.accessVaultClientProvider.APIClient(s.storage)
	if err != nil {
		return "", err
	}
	secret, err := cl.Logical().Read("task/" + task)
	if err != nil {
		return "", err
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("emty response: %#v", secret)
	}
	return parse(secret.Data)
}

func parse(data map[string]interface{}) (string, error) {
	status := data["status"].(string)
	validStatuses := map[string]struct{}{
		"QUEUED": {}, "RUNNING": {}, "SUCCEEDED": {}, "FAILED": {}, "CANCELED": {},
	}
	if _, ok := validStatuses[status]; !ok {
		return "", fmt.Errorf("invalid status: %s", status)
	}
	return status, nil
}
