package flant_gitops

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/logboek"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

var systemClock util.Clock = util.NewSystemClock()

const (
	lastPeriodicRunTimestampKey = "last_periodic_run_timestamp"
)

func (b *backend) PeriodicTask(req *logical.Request) error {
	ctx := context.Background()

	entry, err := req.Storage.Get(ctx, lastPeriodicRunTimestampKey)
	if err != nil {
		return fmt.Errorf("error getting key %q from storage: %s", lastPeriodicRunTimestampKey, err)
	}

	if entry != nil {
		lastRunTimestamp, err := strconv.ParseInt(string(entry.Value), 10, 64)
		// TODO: use fieldNameGitPollPeriod
		if err == nil && systemClock.Since(time.Unix(lastRunTimestamp, 0)) < 5*time.Minute {
			return nil
		}
	}

	now := systemClock.Now()
	uuid, err := b.TaskQueueManager.RunTask(ctx, req.Storage, b.periodicTask)

	if err == queue_manager.QueueBusyError {
		// TODO: use fieldNameGitPollPeriod
		b.Logger().Debug(fmt.Sprintf("Will not add new periodic task: there is currently running task which took more than %s", 5*time.Minute))
		return nil
	}

	if err != nil {
		return fmt.Errorf("error adding queue manager task: %s", err)
	}

	if err := req.Storage.Put(ctx, &logical.StorageEntry{Key: lastPeriodicRunTimestampKey, Value: []byte(fmt.Sprintf("%d", now.Unix()))}); err != nil {
		return fmt.Errorf("error putting last flant gitops run timestamp record by key %q: %s", lastPeriodicRunTimestampKey, err)
	}

	b.LastPeriodicTaskUUID = uuid
	b.Logger().Debug(fmt.Sprintf("Added new periodic task with uuid %s", uuid))

	return nil
}

func (b *backend) periodicTask(ctx context.Context, storage logical.Storage) error {
	b.Logger().Debug("Started periodic task")

	config, err := getConfiguration(ctx, storage)
	if err != nil {
		return err
	} else if config == nil {
		b.Logger().Info("Configuration not set")
		return nil
	}

	b.Logger().Debug(fmt.Sprintf("Got configuration: %+v", config))

	gitCredentials, err := getGitCredential(ctx, storage)
	if err != nil {
		return err
	}

	// clone git repository and get head commit
	var gitRepo *goGit.Repository
	var headCommit string
	{
		b.Logger().Debug(fmt.Sprintf("Cloning git repo %q branch %s", config.GitRepoUrl, config.GitBranchName))

		var cloneOptions trdlGit.CloneOptions
		{
			cloneOptions.BranchName = config.GitBranchName
			cloneOptions.RecurseSubmodules = goGit.DefaultSubmoduleRecursionDepth

			if gitCredentials != nil && gitCredentials.Username != "" && gitCredentials.Password != "" {
				cloneOptions.Auth = &http.BasicAuth{
					Username: gitCredentials.Username,
					Password: gitCredentials.Password,
				}
			}
		}

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
		b.Logger().Debug("Head commit not changed: skipping")
		return nil
	}

	// check that current commit is a descendant of the last successfully processed commit
	if lastSuccessfulCommit != "" {
		isAncestor, err := trdlGit.IsAncestor(gitRepo, lastSuccessfulCommit, headCommit)
		if err != nil {
			return err
		}

		if !isAncestor {
			return fmt.Errorf("unable to run task for git commit %q which is not descendant of the last successfully processed commit %q", headCommit, lastSuccessfulCommit)
		}
	}

	// verify head commit pgp signatures
	{
		trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(ctx, storage)
		if err != nil {
			return fmt.Errorf("unable to get trusted public keys: %s", err)
		}

		requiredNumberOfVerifiedSignaturesOnCommit, err := strconv.Atoi(config.RequiredNumberOfVerifiedSignaturesOnCommit)
		if err != nil {
			return fmt.Errorf("unable to convert %q to int from value: %s", fieldNameRequiredNumberOfVerifiedSignaturesOnCommit, err)
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headCommit, trustedPGPPublicKeys, requiredNumberOfVerifiedSignaturesOnCommit); err != nil {
			return err
		}

		b.Logger().Debug(fmt.Sprintf("Verified %s commit signatures", config.RequiredNumberOfVerifiedSignaturesOnCommit))
	}

	// run docker build with service dockerfile and context
	{
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
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

				if err := docker.GenerateAndAddDockerfileToTar(tw, serviceDockerfilePath, serviceDirInContext, config.DockerImage, []string{config.Command}, false); err != nil {
					return fmt.Errorf("unable to add service dockerfile to tar: %s", err)
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

		b.Logger().Debug(fmt.Sprintf("Running command %q in the base image %q", config.Command, config.DockerImage))

		response, err := cli.ImageBuild(ctx, contextReader, types.ImageBuildOptions{
			NoCache:     true,
			ForceRemove: true,
			PullParent:  true,
			Dockerfile:  serviceDockerfilePath,
			Version:     types.BuilderV1,
		})
		if err != nil {
			return fmt.Errorf("unable to run docker image build: %s", err)
		}

		var outputBuf bytes.Buffer
		out := io.MultiWriter(&outputBuf, logboek.Context(ctx).OutStream())

		if err := docker.DisplayFromImageBuildResponse(out, response); err != nil {
			return fmt.Errorf("error writing docker command output: %s", err)
		}

		b.Logger().Debug("Command output BEGIN\n")
		b.Logger().Debug(outputBuf.String())
		b.Logger().Debug("Command output END\n")

		b.Logger().Debug(fmt.Sprintf("Running command %q in the base image %q DONE", config.Command, config.DockerImage))
	}

	if err := storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKeyLastSuccessfulCommit,
		Value: []byte(headCommit),
	}); err != nil {
		return fmt.Errorf("unable to store last_successful_commit: %s", err)
	}

	return nil
}
