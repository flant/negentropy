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
	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
	uuid "github.com/satori/go.uuid"
	"github.com/werf/logboek"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/shared/client"
)

var systemClock util.Clock = util.NewSystemClock()

const (
	storageKeyLastSuccessfulCommit = "last_successful_commit"
	lastPeriodicRunTimestampKey    = "last_periodic_run_timestamp"
)

func (b *backend) PeriodicTask(req *logical.Request) error {
	ctx := context.Background()

	config, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return fmt.Errorf("unable to get configuration: %s", err)
	}
	if config == nil {
		b.Logger().Info("Configuration not set: skipping periodic task")
		return nil
	}

	{
		cfgData, err := json.MarshalIndent(config, "", "  ")
		b.Logger().Debug(fmt.Sprintf("Got configuration (err=%v):\n%s", err, string(cfgData)))
	}

	gitCredentials, err := trdlGit.GetGitCredential(ctx, req.Storage)
	if err != nil {
		return fmt.Errorf("unable to get Git credentials configuration: %s", err)
	}

	vaultRequests, err := listVaultRequests(ctx, req.Storage)
	if err != nil {
		return fmt.Errorf("unable to get all Vault requests configurations: %s", err)
	}

	for _, cfg := range vaultRequests {
		b.Logger().Debug(fmt.Sprintf("Got configured vault request: %#v\n", cfg))
	}

	apiConfig, err := b.AccessVaultController.GetApiConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to get Vault API config: %s", err)
	}

	if len(vaultRequests) > 0 && apiConfig == nil {
		reqCfgData, _ := json.MarshalIndent(vaultRequests, "", "  ")
		b.Logger().Info(fmt.Sprintf("Vault API access configuration not set, but there are Vault requests configured: skipping periodic task:\n%s\n", reqCfgData))
		return nil
	}

	entry, err := req.Storage.Get(ctx, lastPeriodicRunTimestampKey)
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
	uuid, err := b.TasksManager.RunTask(ctx, req.Storage, func(ctx context.Context, storage logical.Storage) error {
		err := b.periodicTask(ctx, storage, config, gitCredentials, vaultRequests, apiConfig)
		if err != nil {
			b.Logger().Error(fmt.Sprintf("Periodic task failed: %s", err))
		} else {
			b.Logger().Info("Periodic task succeeded")
		}
		return err
	})

	if err == tasks_manager.ErrBusy {
		b.Logger().Debug(fmt.Sprintf("Will not add new periodic task: there is currently running task which took more than %s", config.GitPollPeriod))
		return nil
	}

	if err != nil {
		return fmt.Errorf("unable to add queue manager periodic task: %s", err)
	}

	if err := req.Storage.Put(ctx, &logical.StorageEntry{Key: lastPeriodicRunTimestampKey, Value: []byte(fmt.Sprintf("%d", now.Unix()))}); err != nil {
		return fmt.Errorf("unable to put last periodic task run timestamp in storage by key %q: %s", lastPeriodicRunTimestampKey, err)
	}

	b.LastPeriodicTaskUUID = uuid
	b.Logger().Debug(fmt.Sprintf("Added new periodic task with uuid %s", uuid))

	return nil
}

func (b *backend) periodicTask(ctx context.Context, storage logical.Storage, config *configuration, gitCredentials *trdlGit.GitCredential, vaultRequestsConfig vaultRequests, apiConfig *client.VaultApiConf) error {
	b.Logger().Debug("Started periodic task")

	// clone git repository and get head commit
	var gitRepo *goGit.Repository
	var headCommit string
	{
		b.Logger().Debug(fmt.Sprintf("Cloning git repo %q branch %q", config.GitRepoUrl, config.GitBranch))

		var cloneOptions trdlGit.CloneOptions
		{
			cloneOptions.BranchName = config.GitBranch
			cloneOptions.RecurseSubmodules = goGit.DefaultSubmoduleRecursionDepth

			if gitCredentials != nil && gitCredentials.Username != "" && gitCredentials.Password != "" {
				cloneOptions.Auth = &http.BasicAuth{
					Username: gitCredentials.Username,
					Password: gitCredentials.Password,
				}
			}
		}

		var err error
		if gitRepo, err = trdlGit.CloneInMemory(config.GitRepoUrl, cloneOptions); err != nil {
			return err
		}

		r, err := gitRepo.Head()
		if err != nil {
			return err
		}

		headCommit = r.Hash().String()
		b.Logger().Debug(fmt.Sprintf("Got head commit: %s", headCommit))
	}

	// define lastSuccessfulCommit
	var lastSuccessfulCommit string
	{
		entry, err := storage.Get(ctx, storageKeyLastSuccessfulCommit)
		if err != nil {
			return err
		}

		if entry != nil && string(entry.Value) != "" {
			lastSuccessfulCommit = string(entry.Value)
		} else {
			lastSuccessfulCommit = config.InitialLastSuccessfulCommit
		}

		b.Logger().Debug(fmt.Sprintf("Last successful commit: %s", lastSuccessfulCommit))
	}

	// skip commit if already processed
	if lastSuccessfulCommit == headCommit {
		b.Logger().Debug("Head commit not changed: skipping periodic task")
		return nil
	}

	// check that current commit is a descendant of the last successfully processed commit
	if lastSuccessfulCommit != "" {
		isAncestor, err := trdlGit.IsAncestor(gitRepo, lastSuccessfulCommit, headCommit)
		if err != nil {
			return err
		}

		if !isAncestor {
			return fmt.Errorf("unable to run periodic task for git commit %q which is not descendant of the last successfully processed commit %q", headCommit, lastSuccessfulCommit)
		}
	}

	// verify head commit pgp signatures
	{
		trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(ctx, storage)
		if err != nil {
			return fmt.Errorf("unable to get trusted public keys: %s", err)
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headCommit, trustedPGPPublicKeys, config.RequiredNumberOfVerifiedSignaturesOnCommit); err != nil {
			return err
		}

		b.Logger().Debug(fmt.Sprintf("Verified %d commit signatures", config.RequiredNumberOfVerifiedSignaturesOnCommit))
	}

	vaultRequestEnvs := map[string]string{}
	for _, requestConfig := range vaultRequestsConfig {
		{
			reqData, _ := json.MarshalIndent(requestConfig, "", "  ")
			b.Logger().Debug(fmt.Sprintf("Performing Vault request with configuration:\n%s\n", string(reqData)))
		}

		token, err := b.performWrappedVaultRequest(ctx, requestConfig)
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

				if err := trdlGit.AddWorktreeFilesToTar(tw, gitRepo); err != nil {
					return fmt.Errorf("unable to add git worktree files to tar: %s", err)
				}

				dockerfileOpts := docker.DockerfileOpts{EnvVars: vaultRequestEnvs}

				if apiConfig != nil {
					vaultAddr := apiConfig.APIURL
					vaultCACert := apiConfig.APICa
					vaultCACertPath := path.Join(".flant_gitops", "ca.crt")
					vaultTLSServerName := apiConfig.APIHost

					if err := WriteFilesToTar(tw, map[string][]byte{vaultCACertPath: []byte(vaultCACert + "\n")}); err != nil {
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
					panic(closeErr)
				}
				return
			}

			if err := contextWriter.Close(); err != nil {
				panic(err)
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
		Key:   storageKeyLastSuccessfulCommit,
		Value: []byte(headCommit),
	}); err != nil {
		return fmt.Errorf("unable to store %q: %s", storageKeyLastSuccessfulCommit, err)
	}

	return nil
}

func (b *backend) performWrappedVaultRequest(ctx context.Context, conf *vaultRequest) (string, error) {
	apiClient, err := b.AccessVaultController.APIClient()
	if err != nil {
		return "", fmt.Errorf("unable to get Vault API Client for %q Vault request: %s", conf.Name, err)
	}

	request := apiClient.NewRequest(conf.Method, conf.Path)
	request.WrapTTL = strconv.FormatFloat(conf.WrapTTL.Seconds(), 'f', 0, 64)
	if len(conf.Options) > 0 {
		if err := request.SetJSONBody(conf.Options); err != nil {
			return "", fmt.Errorf("unable to set %q field json data for %q Vault request: %s", fieldNameVaultRequestOptions, conf.Name, err)
		}
	}

	resp, err := apiClient.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf("unable to perform %q Vault request: %s", conf.Name, err)
	}

	defer resp.Body.Close()
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to parse wrap token for %q Vault request: %s", conf.Name, err)
	}

	return secret.WrapInfo.Token, nil
}

func WriteFilesToTar(tw *tar.Writer, filesData map[string][]byte) error {
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
