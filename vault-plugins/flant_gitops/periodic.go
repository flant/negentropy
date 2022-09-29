package flant_gitops

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/git_repository"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/shared/client"
)

var systemClock util.Clock = util.NewSystemClock()

const (
	storageKeyLastStartedCommit    = "last_started_commit"
	storageKeyLastSuccessfulCommit = "last_successful_commit"
	lastPeriodicRunTimestampKey    = "last_periodic_run_timestamp"
)

func (b *backend) PeriodicTask(storage logical.Storage) error {
	ctx := context.Background()
	logger := b.Logger()

	config, err := git_repository.GetConfig(ctx, storage, logger)
	if err != nil {
		return err
	}

	gitCheckintervalExceeded, err := b.checkAndUpdateTimeStamp(ctx, storage, config.GitPollPeriod)

	commitHash, err := git_repository.GitService(ctx, storage, b.Logger()).CheckForNewCommit()
	if err != nil {
		return fmt.Errorf("obtaining new commit: %#w", err)
	}

	if commitHash == nil {
		b.Logger().Debug("No new commits: skipping periodic task")
		return nil
	}

	b.Logger().Info("obtain", "commitHash", commitHash)

	now := systemClock.Now()
	uuid, err := b.TasksManager.RunTask(ctx, storage, func(ctx context.Context, storage logical.Storage) error {
		//err := b.periodicTask(ctx, storage, config, vaultRequests, apiConfig)
		//if err != nil {
		//	logger.Error(fmt.Sprintf("Periodic task failed: %s", err))
		//} else {
		//	logger.Info("Periodic task succeeded")
		//}
		//return err
		return nil
	})

	if err == tasks_manager.ErrBusy {
		logger.Debug(fmt.Sprintf("Will not add new periodic task: there is currently running task which took more than %s", config.GitPollPeriod))
		return nil
	}

	if err != nil {
		return fmt.Errorf("unable to add queue manager periodic task: %s", err)
	}

	if err := storage.Put(ctx, &logical.StorageEntry{Key: lastPeriodicRunTimestampKey, Value: []byte(fmt.Sprintf("%d", now.Unix()))}); err != nil {
		return fmt.Errorf("unable to put last periodic task run timestamp in storage by key %q: %s", lastPeriodicRunTimestampKey, err)
	}

	b.LastPeriodicTaskUUID = uuid
	logger.Debug(fmt.Sprintf("Added new periodic task with uuid %s", uuid))

	return nil
}

func (b *backend) periodicTask(ctx context.Context, storage logical.Storage, config *git_repository.Configuration,
	vaultRequestsConfig interface{}, apiConfig *client.VaultApiConf) error {
	b.Logger().Debug("Started periodic task")

	// verify head commit pgp signatures
	{

	}

	//vaultRequestEnvs := map[string]string{}
	//for _, requestConfig := range vaultRequestsConfig {
	//	{
	//		reqData, _ := json.MarshalIndent(requestConfig, "", "  ")
	//		b.Logger().Debug(fmt.Sprintf("Performing Vault request with Configuration:\n%s\n", string(reqData)))
	//	}
	//
	//	token, err := b.performWrappedVaultRequest(ctx, storage, requestConfig)
	//	if err != nil {
	//		return fmt.Errorf("unable to perform %q Vault request: %s", requestConfig.Name, err)
	//	}
	//
	//	envName := fmt.Sprintf("VAULT_REQUEST_TOKEN_%s", strings.ReplaceAll(strings.ToUpper(requestConfig.Name), "-", "_"))
	//	vaultRequestEnvs[envName] = token
	//
	//	b.Logger().Info(fmt.Sprintf("Performed %q Vault request", requestConfig.Name))
	//}

	// run docker build with service dockerfile and context
	//{
	//	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
	//	if err != nil {
	//		return fmt.Errorf("unable to create docker client: %s", err)
	//	}
	//
	//	serviceDirInContext := ".flant_gitops"
	//	serviceDockerfilePath := path.Join(serviceDirInContext, "Dockerfile")
	//	contextReader, contextWriter := io.Pipe()
	//	go func() {
	//		if err := func() error {
	//			tw := tar.NewWriter(contextWriter)
	//
	//			//if err := trdlGit.AddWorktreeFilesToTar(tw, gitRepo); err != nil {
	//			//	return fmt.Errorf("unable to add git worktree files to tar: %s", err)
	//			//}
	//
	//			dockerfileOpts := docker.DockerfileOpts{EnvVars: vaultRequestEnvs}
	//
	//			if apiConfig != nil {
	//				vaultAddr := apiConfig.APIURL
	//				vaultCACert := apiConfig.CaCert
	//				vaultCACertPath := path.Join(".flant_gitops", "ca.crt")
	//				vaultTLSServerName := apiConfig.APIHost
	//
	//				if err := writeFilesToTar(tw, map[string][]byte{vaultCACertPath: []byte(vaultCACert + "\n")}); err != nil {
	//					return fmt.Errorf("unable to write file %q to tar: %s", vaultCACertPath, err)
	//				}
	//
	//				dockerfileOpts.EnvVars["VAULT_ADDR"] = vaultAddr
	//				dockerfileOpts.EnvVars["VAULT_CACERT"] = path.Join("/git", vaultCACertPath)
	//				dockerfileOpts.EnvVars["VAULT_TLS_SERVER_NAME"] = vaultTLSServerName
	//			}
	//
	//			if err := docker.GenerateAndAddDockerfileToTar(tw, serviceDockerfilePath, config.DockerImage, config.Commands, dockerfileOpts); err != nil {
	//				return fmt.Errorf("unable to add generated Dockerfile to tar: %s", err)
	//			}
	//
	//			if err := tw.Close(); err != nil {
	//				return fmt.Errorf("unable to close tar writer: %s", err)
	//			}
	//
	//			return nil
	//		}(); err != nil {
	//			if closeErr := contextWriter.CloseWithError(err); closeErr != nil {
	//				panic(closeErr) // nolint:panic_check
	//			}
	//			return
	//		}
	//
	//		if err := contextWriter.Close(); err != nil {
	//			panic(err) // nolint:panic_check
	//		}
	//	}()

	//}

	if err := storage.Put(ctx, &logical.StorageEntry{
		Key: storageKeyLastSuccessfulCommit,
		//Value: []byte(headCommit),
	}); err != nil {
		return fmt.Errorf("unable to store %q: %s", storageKeyLastSuccessfulCommit, err)
	}

	return nil
}

// checkAndUpdateTimeStamp returns true if more then interval were spent, and if true: update stored timestamp
func (b *backend) checkAndUpdateTimeStamp(ctx context.Context, storage logical.Storage, interval time.Duration) (bool, error) {
	result := false
	entry, err := storage.Get(ctx, lastPeriodicRunTimestampKey)
	if err != nil {
		return false, fmt.Errorf("unable to get key %q from storage: %w", lastPeriodicRunTimestampKey, err)
	}
	var lastRunTimestamp int64
	if entry != nil {
		// doesn't check parsing error due to nobody write this value - if error should interpret as '0'
		lastRunTimestamp, _ = strconv.ParseInt(string(entry.Value), 10, 64) // nolint: errcheck
	}
	if systemClock.Since(time.Unix(lastRunTimestamp, 0)) > interval {
		result = true
	}
	if result {
		now := time.Now()
		if err := storage.Put(ctx, &logical.StorageEntry{Key: lastPeriodicRunTimestampKey, Value: []byte(fmt.Sprintf("%d", now.Unix()))}); err != nil {
			return result, err
		}
	}
	return result, nil
}
