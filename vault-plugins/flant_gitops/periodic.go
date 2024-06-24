// Main workflow of the flant_gitops.
// Check is new commit signed by specific amount of PGP
// collect vaults with tokens
// run k8s job
// check statuses

// The base of workflow consistency:
// 1) new commit should go through last_started_commit -> last_pushed_to_k8s_commit -> last_k8s_finished_commit before next wil be taken
// 2) new commit should go through last_started_commit -> actually pushed to k8s (not change last_pushed_to_k8s_commit) at one run of PeriodicTask
// 3) changes last_started_commit -> last_pushed_to_k8s_commit -> last_k8s_finished_commit are written only in main goroutine
// 4) job at kube should be eventually terminated (by success/failed/timed out)
// trick: only one place to write data to storage

// Conditions for became new record of last_started_commit:
// 1) last_started_commit = last_pushed_to_k8s_commit = last_k8s_finished_commit
// 2) new  suitable commit at git

// Condition for change last_pushed_to_k8s_commit: at k8s exists job with name of last_pushed_to_k8s_commit

// Conditions for change  last_k8s_finished_commit: Kube has finished job with name last_pushed_to_k8s_commit

// Corner cases:
// 1) Action of pushing to k8s was down without
// 2) Action longs until next periodic task run

// Deal with corner case:
// 1A) Check if job exists at k8s
// 1B) If job not exists - recreate action
// 2) mutex at PeriodicTask

package flant_gitops

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/git_repository"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/kube"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/vault"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
)

// for testability
var (
	systemClock         util.Clock = util.NewSystemClock()
	kubeServiceProvider            = kube.NewKubeService
)

const (
	//  store commit which is taken into work, but
	storageKeyLastStartedCommit     = "last_started_commit"
	storageKeyLastPushedTok8sCommit = "last_pushed_to_k8s_commit"
	storageKeyLastK8sFinishedCommit = "last_k8s_finished_commit"
	lastPeriodicRunTimestampKey     = "last_periodic_run_timestamp"
)

func (b *backend) PeriodicTask(storage logical.Storage) error {
	b.periodicTaskMutex.Lock()
	defer b.periodicTaskMutex.Unlock()

	ctx := context.Background()

	lastStartedCommit, lastPushedToK8sCommit, lastK8sFinishedCommit, err := b.collectSavedWorkingCommits(ctx, storage)
	if err != nil {
		return err
	}

	if lastStartedCommit != lastPushedToK8sCommit {
		b.Logger().Info("corner case detected: lastStartedCommit != lastPushedToK8sCommit")
		err := b.checkAndUpdateLastPushedTok8sCommit(ctx, storage, lastStartedCommit)
		if err != nil {
			return err
		}
		// update values
		_, lastPushedToK8sCommit, lastK8sFinishedCommit, err = b.collectSavedWorkingCommits(ctx, storage)
		if err != nil {
			return err
		}
	}
	println("1111")
	err = b.updateK8sFinishedCommit(ctx, storage, lastPushedToK8sCommit, lastK8sFinishedCommit)
	if err != nil {
		return err
	}
	_, lastPushedToK8sCommit, lastK8sFinishedCommit, err = b.collectSavedWorkingCommits(ctx, storage)
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

// checkAndUpdateLastPushedTok8sCommit check conditions for updating LastPushedTok8sCommit
// check job at of k8s, if pushed -> just LastPushedTok8sCommit
// if not pushed -> rerun action of pushing to k8s
func (b *backend) checkAndUpdateLastPushedTok8sCommit(ctx context.Context, storage logical.Storage, lastStartedCommit string) error {
	kubeService, err := kubeServiceProvider(ctx, storage)
	if err != nil {
		return fmt.Errorf("kubeservice: %w", err)
	}
	exist, _, err := kubeService.CheckJob(ctx, lastStartedCommit)
	if err != nil {
		return fmt.Errorf("kubeservice.checkjob: %w", err)
	}
	if !exist { // corner case
		b.Logger().Warn(fmt.Sprintf("commit %q is not at k8s, repushing job", lastStartedCommit))
		err = b.processCommit(ctx, storage, lastStartedCommit)
		if err != nil {
			return fmt.Errorf("processCommit: %w", err)
		}
	} else {
		b.Logger().Info("commit %q is at k8s")
	}
	return storeLastPushedTok8sCommit(ctx, storage, lastStartedCommit)
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

	err = updateLastRunTimeStamp(ctx, storage, newTimeStamp)
	if err != nil {
		return err
	}

	err = b.processCommit(ctx, storage, *commitHash)
	if err != nil {
		return fmt.Errorf("processCommit: %w", err)
	}
	return nil
}

// checkStatusPushedTok8sCommit checks is pushed commit finished at k8s and returns last finished at k8s commit
func (b *backend) updateK8sFinishedCommit(ctx context.Context, storage logical.Storage, pushedToK8sCommit string, lastK8sFinishedCommit string) error {
	if pushedToK8sCommit == lastK8sFinishedCommit {
		return nil
	}

	kubeService, err := kubeServiceProvider(ctx, storage)
	if err != nil {
		return fmt.Errorf("kubeservice: %w", err)
	}

	_, jobFinished, err := kubeService.CheckJob(ctx, pushedToK8sCommit)
	if err != nil {
		return fmt.Errorf("kubeservice.checkjob: %w", err)
	}

	if jobFinished {
		return storeLastK8sFinishedCommit(ctx, storage, pushedToK8sCommit)
	}
	return nil
}

// collectSavedWorkingCommits gets, checks  and  returns : lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit
// possible valid states: 1)  A, B, B  2) A, A, B 3) A, A, A
func (b *backend) collectSavedWorkingCommits(ctx context.Context, storage logical.Storage) (string, string, string, error) {
	lastStartedCommit, err := util.GetString(ctx, storage, storageKeyLastStartedCommit)
	if err != nil {
		return "", "", "", err
	}
	lastPushedToK8sCommit, err := util.GetString(ctx, storage, storageKeyLastPushedTok8sCommit)
	if err != nil {
		return "", "", "", err
	}
	lastK8sFinishedCommit, err := util.GetString(ctx, storage, storageKeyLastK8sFinishedCommit)
	if err != nil {
		return "", "", "", err
	}
	// checks
	if !(lastPushedToK8sCommit == lastK8sFinishedCommit || lastStartedCommit == lastPushedToK8sCommit) {
		return "", "", "", fmt.Errorf("read wrong combination of working commits: %q, %q, %q",
			lastStartedCommit, lastPushedToK8sCommit, lastK8sFinishedCommit)
	}
	b.Logger().Debug("working commits hashes", "lastStartedCommit", lastStartedCommit,
		"lastPushedToK8sCommit", lastPushedToK8sCommit, "lastK8sFinishedCommit", lastK8sFinishedCommit)
	return lastStartedCommit, lastPushedToK8sCommit, lastK8sFinishedCommit, nil
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
