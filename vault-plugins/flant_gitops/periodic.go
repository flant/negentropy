package flant_gitops

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
	uuid "github.com/satori/go.uuid"
	"github.com/werf/logboek"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/shared/client"
)

var systemClock util.Clock = util.NewSystemClock()

const (
	storageKeyLastSuccessfulCommit = "last_successful_commit"
	lastPeriodicRunTimestampKey    = "last_periodic_run_timestamp"
)

func (b *backend) PeriodicTask(storage logical.Storage) error {
	ctx := context.Background()
	logger := b.Logger()

	config, err := getConfig(ctx, storage, logger)
	if err != nil {
		return err
	}

	vaultRequests, err := listVaultRequests(ctx, storage)
	if err != nil {
		return fmt.Errorf("unable to get all Vault requests configurations: %s", err)
	}

	for _, cfg := range vaultRequests {
		logger.Debug(fmt.Sprintf("Got configured vault request: %#v\n", cfg))
	}

	apiConfig, err := b.AccessVaultClientProvider.GetApiConfig(ctx, storage)
	if err != nil {
		return fmt.Errorf("unable to get Vault API config: %s", err)
	}

	if len(vaultRequests) > 0 && apiConfig == nil {
		reqCfgData, _ := json.MarshalIndent(vaultRequests, "", "  ")
		b.Logger().Info(fmt.Sprintf("Vault API access configuration not set, but there are Vault requests configured: skipping periodic task:\n%s\n", reqCfgData))
		return nil
	}

	entry, err := storage.Get(ctx, lastPeriodicRunTimestampKey)
	if err != nil {
		return fmt.Errorf("unable to get key %q from storage: %s", lastPeriodicRunTimestampKey, err)
	}

	if entry != nil {
		lastRunTimestamp, err := strconv.ParseInt(string(entry.Value), 10, 64)
		if err == nil && systemClock.Since(time.Unix(lastRunTimestamp, 0)) < config.GitPollPeriod {
			b.Logger().Debug("Waiting Git poll period: skipping periodic task")
			return nil
		}
	}

	now := systemClock.Now()
	uuid, err := b.TasksManager.RunTask(ctx, storage, func(ctx context.Context, storage logical.Storage) error {
		err := b.periodicTask(ctx, storage, config, vaultRequests, apiConfig)
		if err != nil {
			logger.Error(fmt.Sprintf("Periodic task failed: %s", err))
		} else {
			logger.Info("Periodic task succeeded")
		}
		return err
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

func (b *backend) periodicTask(ctx context.Context, storage logical.Storage, config *configuration,
	vaultRequestsConfig vaultRequests, apiConfig *client.VaultApiConf) error {
	b.Logger().Debug("Started periodic task")

	// verify head commit pgp signatures
	{

	}

	vaultRequestEnvs := map[string]string{}
	for _, requestConfig := range vaultRequestsConfig {
		{
			reqData, _ := json.MarshalIndent(requestConfig, "", "  ")
			b.Logger().Debug(fmt.Sprintf("Performing Vault request with configuration:\n%s\n", string(reqData)))
		}

		token, err := b.performWrappedVaultRequest(ctx, storage, requestConfig)
		if err != nil {
			return fmt.Errorf("unable to perform %q Vault request: %s", requestConfig.Name, err)
		}

		envName := fmt.Sprintf("VAULT_REQUEST_TOKEN_%s", strings.ReplaceAll(strings.ToUpper(requestConfig.Name), "-", "_"))
		vaultRequestEnvs[envName] = token

		b.Logger().Info(fmt.Sprintf("Performed %q Vault request", requestConfig.Name))
	}

	// run docker build with service dockerfile and context
	{
		cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
		if err != nil {
			return fmt.Errorf("unable to create docker client: %s", err)
		}

		serviceDirInContext := ".flant_gitops"
		serviceDockerfilePath := path.Join(serviceDirInContext, "Dockerfile")
		contextReader, contextWriter := io.Pipe()
		go func() {
			if err := func() error {
				tw := tar.NewWriter(contextWriter)

				//if err := trdlGit.AddWorktreeFilesToTar(tw, gitRepo); err != nil {
				//	return fmt.Errorf("unable to add git worktree files to tar: %s", err)
				//}

				dockerfileOpts := docker.DockerfileOpts{EnvVars: vaultRequestEnvs}

				if apiConfig != nil {
					vaultAddr := apiConfig.APIURL
					vaultCACert := apiConfig.CaCert
					vaultCACertPath := path.Join(".flant_gitops", "ca.crt")
					vaultTLSServerName := apiConfig.APIHost

					if err := writeFilesToTar(tw, map[string][]byte{vaultCACertPath: []byte(vaultCACert + "\n")}); err != nil {
						return fmt.Errorf("unable to write file %q to tar: %s", vaultCACertPath, err)
					}

					dockerfileOpts.EnvVars["VAULT_ADDR"] = vaultAddr
					dockerfileOpts.EnvVars["VAULT_CACERT"] = path.Join("/git", vaultCACertPath)
					dockerfileOpts.EnvVars["VAULT_TLS_SERVER_NAME"] = vaultTLSServerName
				}

				if err := docker.GenerateAndAddDockerfileToTar(tw, serviceDockerfilePath, config.DockerImage, config.Commands, dockerfileOpts); err != nil {
					return fmt.Errorf("unable to add generated Dockerfile to tar: %s", err)
				}

				if err := tw.Close(); err != nil {
					return fmt.Errorf("unable to close tar writer: %s", err)
				}

				return nil
			}(); err != nil {
				if closeErr := contextWriter.CloseWithError(err); closeErr != nil {
					panic(closeErr) // nolint:panic_check
				}
				return
			}

			if err := contextWriter.Close(); err != nil {
				panic(err) // nolint:panic_check
			}
		}()

		b.Logger().Debug(fmt.Sprintf("Running commands %+q in the base image %q", config.Commands, config.DockerImage))

		serviceLabels := map[string]string{
			"negentropy-flant-gitops-periodic-uuid": uuid.NewV4().String(),
		}

		response, err := cli.ImageBuild(ctx, contextReader, types.ImageBuildOptions{
			Dockerfile:  serviceDockerfilePath,
			Labels:      serviceLabels,
			NoCache:     true,
			ForceRemove: true,
			PullParent:  true,
			Version:     types.BuilderV1,
		})
		if err != nil {
			return fmt.Errorf("unable to run docker image build: %s", err)
		}

		var outputBuf bytes.Buffer
		out := io.MultiWriter(&outputBuf, logboek.Context(ctx).OutStream())

		if err := docker.DisplayFromImageBuildResponse(out, response); err != nil {
			return fmt.Errorf("error writing Docker command output: %s", err)
		}

		b.Logger().Debug("Command output BEGIN\n")
		b.Logger().Debug(outputBuf.String())
		b.Logger().Debug("Command output END\n")

		b.Logger().Debug(fmt.Sprintf("Commands %+q in the base image %q succeeded", config.Commands, config.DockerImage))

		if err := docker.RemoveImagesByLabels(ctx, cli, serviceLabels); err != nil {
			return fmt.Errorf("unable to remove service docker image: %s", err)
		}
	}

	if err := storage.Put(ctx, &logical.StorageEntry{
		Key: storageKeyLastSuccessfulCommit,
		//Value: []byte(headCommit),
	}); err != nil {
		return fmt.Errorf("unable to store %q: %s", storageKeyLastSuccessfulCommit, err)
	}

	return nil
}

func (b *backend) performWrappedVaultRequest(ctx context.Context, storage logical.Storage,
	vaultReq *vaultRequest) (string, error) {
	apiClient, err := b.AccessVaultClientProvider.APIClient(storage)
	if err != nil {
		return "", fmt.Errorf("unable to get Vault API Client for %q Vault request: %s", vaultReq.Name, err)
	}

	request := apiClient.NewRequest(vaultReq.Method, vaultReq.Path)
	request.WrapTTL = strconv.FormatFloat(vaultReq.WrapTTL.Seconds(), 'f', 0, 64)
	if len(vaultReq.Options) > 0 {
		if err := request.SetJSONBody(vaultReq.Options); err != nil {
			return "", fmt.Errorf("unable to set %q field json data for %q Vault request: %s",
				fieldNameVaultRequestOptions, vaultReq.Name, err)
		}
	}

	resp, err := apiClient.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf("unable to perform %q Vault request: %s", vaultReq.Name, err)
	}

	defer resp.Body.Close()
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to parse wrap token for %q Vault request: %s", vaultReq.Name, err)
	}

	return secret.WrapInfo.Token, nil
}

func writeFilesToTar(tw *tar.Writer, filesData map[string][]byte) error {
	for path, data := range filesData {
		header := &tar.Header{
			Format:     tar.FormatGNU,
			Name:       path,
			Size:       int64(len(data)),
			Mode:       int64(os.ModePerm),
			ModTime:    time.Now(),
			AccessTime: time.Now(),
			ChangeTime: time.Now(),
		}

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("unable to write tar entry %q header: %s", path, err)
		}

		if _, err := tw.Write(data); err != nil {
			return fmt.Errorf("unable to write tar entry %q data: %s", path, err)
		}
	}

	return nil
}
