package task_manager

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/shared/client"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
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
	accessVaultClientProvider client.AccessVaultClientController
	logger                    hclog.Logger
}

func Service(storage logical.Storage, accessVaultClientProvider client.AccessVaultClientController, parentLogger hclog.Logger) TaskService {
	return &service{
		storage:                   storage,
		accessVaultClientProvider: accessVaultClientProvider,
		logger:                    parentLogger.Named("task_server"),
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
	return checkTask(ctx, s.storage, commit, s.readTaskStatus, s.logger)
}

func checkTask(ctx context.Context, storage logical.Storage, commit hashCommit,
	taskStatusProvider func(task taskUUID) (string, error), logger hclog.Logger) (taskExist, taskIsFinished, error) {
	logger.Debug("start checking task status for commit", "hash_commit", commit)
	tasks, err := util.GetStringMap(ctx, storage, storageKeyTasks)
	if err != nil || tasks == nil {
		return false, false, fmt.Errorf("getting tasks from storage: %w", err)
	}
	task, ok := tasks[commit]
	if !ok {
		return false, false, nil
	}
	status, err := taskStatusProvider(task)
	if errors.Is(err, consts.ErrNotFound) { // task may be created but have no statuses
		return true, false, nil
	}
	if err != nil || tasks == nil {
		return true, false, fmt.Errorf("getting task %q status: %w", task, err)
	}
	logger.Debug("result checking task status for hash", "hash_commit", commit, "task_uuid",
		task, "status", status)
	finishedStatuses := map[string]struct{}{"SUCCEEDED": {}, "FAILED": {}, "CANCELED": {}}
	if _, ok := finishedStatuses[status]; ok {
		return true, true, nil
	}
	return true, false, nil
}

func (s *service) readTaskStatus(task taskUUID) (string, error) {
	cl, err := s.accessVaultClientProvider.APIClient()
	if err != nil {
		return "", err
	}
	secret, err := cl.Logical().Read("gitops/task/" + task)
	if err != nil {
		return "", err
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("empty response: %#v", secret)
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
