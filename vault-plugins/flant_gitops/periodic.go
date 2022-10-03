// Main workflow of the flant_gitops.
// Check is new commit signed by specific amount of PGP
// collect vaults with tokens
// run k8s job
// check statuses

package flant_gitops

import (
	"context"
	"fmt"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/kube"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/git_repository"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/vault"
)

var systemClock util.Clock = util.NewSystemClock()

const (
	//  store commit which is taken into work, but
	storageKeyLastStartedCommit     = "last_started_commit"
	storageKeyLastPushedTok8sCommit = "last_pushed_to_k8s_commit"
	storageKeyLastK8sFinishedCommit = "last_k8s_finished_commit"
	lastPeriodicRunTimestampKey     = "last_periodic_run_timestamp"
)

func (b *backend) PeriodicTask(storage logical.Storage) error {
	ctx := context.Background()
	logger := b.Logger()

	lastStartedCommit, lastPushedToK8sCommit, lastK8sFinishedCommit, err := collectWorkingCommits(ctx, storage)
	if err != nil {
		return err
	}

	// TODO probably need to check status of last run task to restart it in case of fail

	logger.Info("got working commits hashes:", lastStartedCommit, lastPushedToK8sCommit, lastK8sFinishedCommit)

	if lastStartedCommit != lastPushedToK8sCommit {
		logger.Info(fmt.Sprintf("commit %s is still not pushed to k8s, skipping periodic function", lastStartedCommit))
		return nil
	}

	lastK8sFinishedCommit, err = b.updateK8sFinishedCommit(ctx, storage, lastPushedToK8sCommit, lastK8sFinishedCommit)
	if err != nil {
		return err
	}

	if lastK8sFinishedCommit != lastPushedToK8sCommit {
		logger.Info(fmt.Sprintf("commit %s is still not finished at k8s, skipping periodic function", lastPushedToK8sCommit))
		return nil
	}

	logger.Info(fmt.Sprintf("commit %s is finished at k8s, continue periodic function...", lastPushedToK8sCommit))

	return b.processGit(ctx, storage, lastPushedToK8sCommit)
}

func (b *backend) processGit(ctx context.Context, storage logical.Storage, lastPushedToK8sCommit string) error {
	logger := b.Logger()
	config, err := git_repository.GetConfig(ctx, storage, logger)
	if err != nil {
		return err
	}

	gitCheckintervalExceeded, err := checkExceedingInterval(ctx, storage, config.GitPollPeriod)
	if err != nil {
		return err
	}

	if !gitCheckintervalExceeded {
		logger.Info("git poll interval not exceeded, finish periodic task")
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

	b.Logger().Info("obtain", "commitHash", commitHash)

	uuid, err := b.TasksManager.RunTask(ctx, storage, func(ctx context.Context, storage logical.Storage) error {
		return b.processCommit(ctx, storage, *commitHash)
	})

	if err != nil {
		return fmt.Errorf("unable to add queue manager periodic task: %s", err)
	}

	logger.Debug(fmt.Sprintf("Added new periodic task with uuid %s", uuid))

	return updateLastRunTimeStamp(ctx, storage, newTimeStamp)
}

// checkStatusPushedTok8sCommit checks is pushed commit finished at k8s and returns last finished at k8s commit
func (b *backend) updateK8sFinishedCommit(ctx context.Context, storage logical.Storage, pushedToK8sCommit string, lastK8sFinishedCommit string) (string, error) {
	if pushedToK8sCommit == lastK8sFinishedCommit {
		return lastK8sFinishedCommit, nil
	}
	var isFinished bool

	kubeService, err := kube.NewKubeService(ctx, storage)
	if err != nil {
		return lastK8sFinishedCommit, err
	}
	isFinished, err = kubeService.IsJobFinished(ctx, pushedToK8sCommit)
	if err != nil {
		return lastK8sFinishedCommit, err
	}

	if isFinished {
		err := util.PutString(ctx, storage, storageKeyLastK8sFinishedCommit, pushedToK8sCommit)
		if err != nil {
			return lastK8sFinishedCommit, err
		}
		lastK8sFinishedCommit = pushedToK8sCommit
	}
	return lastK8sFinishedCommit, nil
}

// collectWorkingCommits gets, checks  and  returns : lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit
// possible valid states: 1)  A, B, B  2) A, A, B 3) A, A, A
func collectWorkingCommits(ctx context.Context, storage logical.Storage) (string, string, string, error) {
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
		return "", "", "", fmt.Errorf("read wrong combination of working commits: %s, %s, %s",
			lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit)
	}
	return lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, nil
}

// checkExceedingInterval returns true if more then interval were spent
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

// processCommit
func (b *backend) processCommit(ctx context.Context, storage logical.Storage, hashCommit string) error {
	apiClient, err := b.AccessVaultClientProvider.APIClient(storage)
	if err != nil {
		return err
	}
	vaultsEnvBase64Json, warnings, err := vault.BuildVaultsBase64Env(ctx, storage, apiClient)
	if len(warnings) > 0 {
		for _, w := range warnings {
			b.Logger().Warn(w)
		}
	}
	b.Logger().Debug("TODO REMOVE", "vaults", vaultsEnvBase64Json)
	kubeService, err := kube.NewKubeService(ctx, storage)
	if err != nil {
		return err
	}
	err = kubeService.RunJob(ctx, hashCommit, vaultsEnvBase64Json)
	if err != nil {
		return err
	}
	b.Logger().Debug("TODO REWRITE!", "jobName", hashCommit)
	return nil
}
