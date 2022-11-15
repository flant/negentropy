// Main workflow of the flant_gitops.
// Check is new commit signed by specific amount of PGP
// collect vaults with tokens
// run k8s job
// check statuses

// The base of workflow consistency:
// 1) new commit should go through last_started_commit -> {task for commit at task_manager} -> last_pushed_to_k8s_commit -> last_k8s_finished_commit
// 2) changes last_started_commit -> last_pushed_to_k8s_commit -> last_k8s_finished_commit are written only in main goroutine
// 3) action of created by commit task should finish as succeeded or failed
// 4) job at kube should be eventually terminated (by success/failed/timed out)
// trick: only one place to write data to storage

// Conditions for became new record of last_started_commit:
// 1) last_started_commit = last_pushed_to_k8s_commit = last_k8s_finished_commit
// 2) new  suitable commit at git

// Conditions for change last_pushed_to_k8s_commit:
// A) Normal task finish
// A1) task for last_started_commit is finished with any status (SUCCEEDED/FAILED/CANCELED)
// A2) kube has job with name last_started_commit
// B) Abnormal task finish
// B1) task for last_started_commit is finished with any status (SUCCEEDED/FAILED/CANCELED)
// B2) kube doesn't have job with name last_started_commit

// Conditions for change  last_k8s_finished_commit:
// A) Normal flow
// A1) Kube has finished job with name last_pushed_to_k8s_commit
// B) Abnormal flow
// B1) Kube doesn't have job with name last_pushed_to_k8s_commit and has finished task for commit last_pushed_to_k8s_commit

// Corner cases:
// 1) No task for last_started_commit (it can happen if vault was downed until task_manager returns uuid of placed task, or periodic function was late to store)
// 2) ?

// Deal with corner case: retry create task and finish periodic function
// If it will be dublicate, it is just fail on attempts to place job with the same name at kube

package flant_gitops

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/vault/sdk/logical"
	trdl_task_manager "github.com/werf/trdl/server/pkg/tasks_manager"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/git_repository"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/kube"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/task_manager"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/vault"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
)

// for testability
var (
	systemClock                util.Clock = util.NewSystemClock()
	kubeServiceProvider                   = kube.NewKubeService
	taskManagerServiceProvider            = task_manager.Service
)

const (
	//  store commit which is taken into work, but
	storageKeyLastStartedCommit     = "last_started_commit"
	storageKeyLastPushedTok8sCommit = "last_pushed_to_k8s_commit"
	storageKeyLastK8sFinishedCommit = "last_k8s_finished_commit"
	lastPeriodicRunTimestampKey     = "last_periodic_run_timestamp"
)

func (b *backend) PeriodicTask(storage logical.Storage) error {
	ctx := context.Background()

	lastStartedCommit, lastPushedToK8sCommit, lastK8sFinishedCommit, err := collectSavedWorkingCommits(ctx, storage)
	if err != nil {
		return err
	}

	b.Logger().Info("got working commits hashes", "lastStartedCommit", lastStartedCommit,
		"lastPushedToK8sCommit", lastPushedToK8sCommit, "lastK8sFinishedCommit", lastK8sFinishedCommit)

	if lastStartedCommit != lastPushedToK8sCommit {
		// check conditions for change lastPushedTok8sCommit
		cornerCase, isLastPushedCommitChanged, err := b.updateLastPushedTok8sCommit(ctx, storage, lastStartedCommit)
		if err != nil {
			return err
		}
		if cornerCase || !isLastPushedCommitChanged { // corner case or still  lastStartedCommit != lastPushedToK8sCommit
			return nil
		}
		// update values
		_, lastPushedToK8sCommit, lastK8sFinishedCommit, err = collectSavedWorkingCommits(ctx, storage)
		if err != nil {
			return err
		}
	}

	err = b.updateK8sFinishedCommit(ctx, storage, lastPushedToK8sCommit, lastK8sFinishedCommit)
	if err != nil {
		return err
	}
	_, lastPushedToK8sCommit, lastK8sFinishedCommit, err = collectSavedWorkingCommits(ctx, storage)
	if err != nil {
		return err
	}

	if lastK8sFinishedCommit != lastPushedToK8sCommit {
		b.Logger().Info(fmt.Sprintf("commit %q is still not finished at k8s, skipping periodic function", lastPushedToK8sCommit))
		return nil
	}

	b.Logger().Info(fmt.Sprintf("commit %q is finished at k8s, continue periodic function...", lastPushedToK8sCommit))

	return b.processGit(ctx, storage, lastK8sFinishedCommit)
}

type (
	isCornerCase = bool
	isPushed     = bool
)

// updateLastPushedTok8sCommit check conditions for updating LastPushedTok8sCommit, returns isCornerCase and isPushed
func (b *backend) updateLastPushedTok8sCommit(ctx context.Context, storage logical.Storage, lastStartedCommit string) (isCornerCase, isPushed, error) {
	exist, finished, err := taskManagerServiceProvider(storage, b.AccessVaultClientProvider, b.Logger()).CheckTask(ctx, lastStartedCommit)
	if err != nil {
		return false, false, err
	}
	if !exist { // corner case: unexpected vault crash happens: recreate task for lastStartedCommit
		b.Logger().Warn(fmt.Sprintf("commit %q has no task, recreate tsk, and interrupt periodic function", lastStartedCommit))
		err = b.createTask(ctx, storage, lastStartedCommit)
		return true, false, err
	}
	if !finished {
		b.Logger().Info(fmt.Sprintf("commit %q is still not pushed to k8s, skipping periodic function", lastStartedCommit))
		return false, false, nil
	}
	// task is finished, no matter is job at k8s or not: change  last_pushed_to_k8s_commit
	b.Logger().Info(fmt.Sprintf("task run by commit %q is finished", lastStartedCommit))

	return false, true, storeLastPushedTok8sCommit(ctx, storage, lastStartedCommit)
}

func (b *backend) processGit(ctx context.Context, storage logical.Storage, lastPushedToK8sCommit string) error {
	config, err := git_repository.GetConfig(ctx, storage, b.Logger())
	if err != nil {
		return err
	}

	gitCheckintervalExceeded, err := checkExceedingInterval(ctx, storage, config.GitPollPeriod)
	if err != nil {
		return err
	}

	if !gitCheckintervalExceeded {
		b.Logger().Info("git poll interval not exceeded, finish periodic task")
		return nil
	}

	newTimeStamp := systemClock.Now()
	commitHash, err := git_repository.GitService(ctx, storage, b.Logger()).CheckForNewCommitFrom(lastPushedToK8sCommit)
	if err != nil {
		return fmt.Errorf("obtaining new commit: %w", err)
	}

	if commitHash == nil {
		b.Logger().Debug("No new commits: finish periodic task")
		return nil
	}
	b.Logger().Info("obtain", "commitHash", *commitHash)

	if err := storeLastStartedCommit(ctx, storage, *commitHash); err != nil {
		return err
	}

	err = b.createTask(ctx, storage, *commitHash)
	if err != nil {
		return err
	}

	return updateLastRunTimeStamp(ctx, storage, newTimeStamp)
}

// createTask creates task and store gotten task_uuid
func (b *backend) createTask(ctx context.Context, storage logical.Storage, commitHash string) error {
	taskUUID, err := b.TasksManager.RunTask(ctx, storage, func(ctx context.Context, storage logical.Storage) error {
		return b.processCommit(ctx, storage, commitHash)
	})
	if errors.Is(err, trdl_task_manager.ErrBusy) {
		b.Logger().Warn(fmt.Sprintf("unable to add queue manager task: %s", err.Error()))
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to add queue manager task: %w", err)
	}

	b.Logger().Debug(fmt.Sprintf("Added new task with uuid %q for commitHash: %q", taskUUID, commitHash))
	return taskManagerServiceProvider(storage, b.AccessVaultClientProvider, b.Logger()).SaveTask(ctx, taskUUID, commitHash)
}

// checkStatusPushedTok8sCommit checks is pushed commit finished at k8s and returns last finished at k8s commit
func (b *backend) updateK8sFinishedCommit(ctx context.Context, storage logical.Storage, pushedToK8sCommit string, lastK8sFinishedCommit string) error {
	if pushedToK8sCommit == lastK8sFinishedCommit {
		return nil
	}
	_, taskFinished, err := taskManagerServiceProvider(storage, b.AccessVaultClientProvider, b.Logger()).CheckTask(ctx, pushedToK8sCommit)
	if err != nil {
		return err
	}

	kubeService, err := kubeServiceProvider(ctx, storage)
	if err != nil {
		return err
	}

	jobExist, jobFinished, err := kubeService.CheckJob(ctx, pushedToK8sCommit)
	if err != nil {
		return err
	}

	if (taskFinished && !jobExist) || jobFinished {
		return storeLastK8sFinishedCommit(ctx, storage, pushedToK8sCommit)
	}

	return nil
}

// collectSavedWorkingCommits gets, checks  and  returns : lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit
// possible valid states: 1)  A, B, B  2) A, A, B 3) A, A, A
func collectSavedWorkingCommits(ctx context.Context, storage logical.Storage) (string, string, string, error) {
	lastStartedCommit, err := util.GetString(ctx, storage, storageKeyLastStartedCommit)
	if err != nil {
		return "", "", "", err
	}
	lastPushedToK8sCommit, err := util.GetString(ctx, storage, storageKeyLastPushedTok8sCommit)
	if err != nil {
		return "", "", "", err
	}
	LastK8sFinishedCommit, err := util.GetString(ctx, storage, storageKeyLastK8sFinishedCommit)
	if err != nil {
		return "", "", "", err
	}
	// checks
	if !(lastPushedToK8sCommit == LastK8sFinishedCommit || lastStartedCommit == lastPushedToK8sCommit) {
		return "", "", "", fmt.Errorf("read wrong combination of working commits: %q, %q, %q",
			lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit)
	}
	return lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, nil
}

// checkExceedingInterval returns true if more than interval were spent
func checkExceedingInterval(ctx context.Context, storage logical.Storage, interval time.Duration) (bool, error) {
	result := false
	lastRunTimestamp, err := util.GetInt64(ctx, storage, lastPeriodicRunTimestampKey)
	if err != nil {
		return false, err
	}
	if systemClock.Since(time.Unix(lastRunTimestamp, 0)) > interval {
		result = true
	}
	return result, nil
}

func updateLastRunTimeStamp(ctx context.Context, storage logical.Storage, timeStamp time.Time) error {
	return util.PutInt64(ctx, storage, lastPeriodicRunTimestampKey, timeStamp.Unix())
}

// processCommit aim action with retries
func (b *backend) processCommit(ctx context.Context, storage logical.Storage, hashCommit string) error {
	// there are retry inside BuildVaultsBase64Env
	apiClient, err := b.AccessVaultClientProvider.APIClient()
	if err != nil {
		return err
	}
	vaultsEnvBase64Json, warnings, err := vault.BuildVaultsBase64Env(ctx, storage, apiClient, b.Logger())
	if len(warnings) > 0 {
		for _, w := range warnings {
			b.Logger().Warn(w)
		}
	}
	b.Logger().Debug("BuildVaultsBase64Env", "vaultsEnvBase64Json", vaultsEnvBase64Json)
	err = backoff.Retry(func() error {
		kubeService, err := kubeServiceProvider(ctx, storage)
		if err != nil {
			return err
		}
		return kubeService.RunJob(ctx, hashCommit, vaultsEnvBase64Json, b.Logger())
	}, sharedio.TwoMinutesBackoff())
	return err
}

func storeLastStartedCommit(ctx context.Context, storage logical.Storage, hashCommit string) error {
	return util.PutString(ctx, storage, storageKeyLastStartedCommit, hashCommit)
}

func storeLastPushedTok8sCommit(ctx context.Context, storage logical.Storage, hashCommit string) error {
	return util.PutString(ctx, storage, storageKeyLastPushedTok8sCommit, hashCommit)
}

func storeLastK8sFinishedCommit(ctx context.Context, storage logical.Storage, hashCommit string) error {
	return util.PutString(ctx, storage, storageKeyLastK8sFinishedCommit, hashCommit)
}
