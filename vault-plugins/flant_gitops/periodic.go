package flant_gitops

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
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

func GetPeriodicTaskFunc(b *backend) func(context.Context, *logical.Request) error {
	return func(_ context.Context, req *logical.Request) error {
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
}

func (b *backend) periodicTask(ctx context.Context, storage logical.Storage) error {
	b.Logger().Debug("Started periodic task")

	fields, err := b.getConfiguration(ctx, storage)
	if err != nil {
		b.Logger().Debug(fmt.Sprintf("Get configuration failed: %s", err))
		return err
	}

	b.Logger().Debug(fmt.Sprintf("Got configuration fields: %#v", fields))

	getRequiredConfigurationFieldFunc := func(fieldName string) (interface{}, error) {
		val, ok := fields.GetOk(fieldName)
		if !ok {
			return nil, fmt.Errorf("invalid configuration in storage: the field %q must be set", fieldName)
		}

		return val, nil
	}

	// define git credential
	var gitUsername string
	var gitPassword string
	{
		gitCredential, err := getGitCredential(ctx, storage)
		if err != nil {
			return err
		}

		if gitCredential != nil {
			gitUsername = gitCredential.Username
			gitPassword = gitCredential.Password
		}
	}

	// clone git repository and get head commit
	var gitRepo *goGit.Repository
	var headCommit string
	{
		repoUrl, err := getRequiredConfigurationFieldFunc(fieldNameGitRepoUrl)
		if err != nil {
			return err
		}

		branchName, err := getRequiredConfigurationFieldFunc(fieldNameGitBranch)
		if err != nil {
			return err
		}

		b.Logger().Debug(fmt.Sprintf("Cloning git repo %q branch %s", repoUrl, branchName))

		var cloneOptions trdlGit.CloneOptions
		{
			cloneOptions.BranchName = branchName.(string)
			cloneOptions.RecurseSubmodules = goGit.DefaultSubmoduleRecursionDepth

			if gitUsername != "" && gitPassword != "" {
				cloneOptions.Auth = &http.BasicAuth{
					Username: gitUsername,
					Password: gitPassword,
				}
			}
		}

		if gitRepo, err = trdlGit.CloneInMemory(repoUrl.(string), cloneOptions); err != nil {
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

		var lastSuccessfulCommit string
		if entry != nil && string(entry.Value) != "" {
			lastSuccessfulCommit = string(entry.Value)
		} else {
			lastSuccessfulCommit = fields.Get(fieldNameInitialLastSuccessfulCommit).(string)
		}

		hclog.L().Debug(fmt.Sprintf("Last successful commit: %s", lastSuccessfulCommit))
	}

	// skip commit if already processed
	if lastSuccessfulCommit == headCommit {
		hclog.L().Debug("Head commit not changed: skipping")
		return nil
	}

	// check that current commit is a descendant of the last successfully processed commit
	if lastSuccessfulCommit != "" {
		isAncestor, err := trdlGit.IsAncestor(gitRepo, lastSuccessfulCommit, headCommit)
		if err != nil {
			return err
		}

		if !isAncestor {
			return fmt.Errorf("unable to run task for git commit %q which is not desendant of the last successfully processed commit %q", headCommit, lastSuccessfulCommit)
		}
	}

	// verify head commit pgp signatures
	{
		requiredNumberOfVerifiedSignaturesOnCommit, err := getRequiredConfigurationFieldFunc(fieldNameRequiredNumberOfVerifiedSignaturesOnCommit)
		if err != nil {
			return err
		}

		trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(ctx, storage)
		if err != nil {
			return fmt.Errorf("unable to get trusted public keys: %s", err)
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headCommit, trustedPGPPublicKeys, requiredNumberOfVerifiedSignaturesOnCommit.(int)); err != nil {
			return err
		}

		b.Logger().Debug(fmt.Sprintf("Verified %d commit signatures", requiredNumberOfVerifiedSignaturesOnCommit))
	}

	// run docker build with service dockerfile and context
	{
		dockerImage, err := getRequiredConfigurationFieldFunc(fieldNameDockerImage)
		if err != nil {
			return err
		}

		command, err := getRequiredConfigurationFieldFunc(fieldNameCommand)
		if err != nil {
			return err
		}

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

				if err := docker.GenerateAndAddDockerfileToTar(tw, serviceDockerfilePath, serviceDirInContext, dockerImage.(string), []string{command.(string)}, false); err != nil {
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

		b.Logger().Debug(fmt.Sprintf("Running command %q in the base image %q", command, dockerImage))

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

		if err := docker.DisplayFromImageBuildResponse(logboek.Context(ctx).OutStream(), response); err != nil {
			return err
		}

		b.Logger().Debug(fmt.Sprintf("Running command %q in the base image %q DONE", command, dockerImage))
	}

	if err := storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKeyLastSuccessfulCommit,
		Value: []byte(headCommit),
	}); err != nil {
		return fmt.Errorf("unable to store last_successful_commit: %s", err)
	}

	return nil
}

func (b *backend) getConfiguration(ctx context.Context, storage logical.Storage) (*framework.FieldData, error) {
	entry, err := storage.Get(ctx, storageKeyConfiguration)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, fmt.Errorf("no configuration found in storage")
	}

	data := make(map[string]interface{})
	if err := json.Unmarshal(entry.Value, &data); err != nil {
		return nil, err
	}

	b.Logger().Debug(fmt.Sprintf("Unmarshalled json: %s", entry.Value))

	fields := &framework.FieldData{}
	fields.Raw = data
	fields.Schema = b.getConfigureFieldSchemaMap()

	return fields, nil
}

func (b *backend) getConfigureFieldSchemaMap() map[string]*framework.FieldSchema {
	for _, p := range b.Paths {
		if p.Pattern == pathPatternConfigure {
			return p.Fields
		}
	}

	b.Logger().Debug(fmt.Sprintf("Unexpected configuration, no path has matched pathPatternConfigure=%q", pathPatternConfigure))

	panic("runtime error")
}
