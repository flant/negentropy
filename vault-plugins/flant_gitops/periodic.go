package flant_gitops

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	goGit "github.com/go-git/go-git/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/logboek"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
)

func (b *backend) periodicTask(ctx context.Context, storage logical.Storage) error {
	hclog.L().Debug(fmt.Sprintf("Started periodic task"))

	fields, err := b.getConfiguration(ctx, storage)
	if err != nil {
		hclog.L().Debug(fmt.Sprintf("Get configuration failed: %s", err))
		return err
	}

	hclog.L().Debug(fmt.Sprintf("Got configuration fields"))

	getRequiredConfigurationFieldFunc := func(fieldName string) (interface{}, error) {
		val, ok := fields.GetOk(fieldName)
		if !ok {
			return nil, fmt.Errorf("invalid configuration in storage: the field %q must be set", fieldName)
		}

		return val, nil
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

		hclog.L().Debug(fmt.Sprintf("Cloning git repo %q branch %s", repoUrl, branchName))

		if gitRepo, err = trdlGit.CloneInMemory(repoUrl.(string), trdlGit.CloneOptions{
			BranchName:        branchName.(string),
			RecurseSubmodules: goGit.DefaultSubmoduleRecursionDepth,
		}); err != nil {
			return err
		}

		r, err := gitRepo.Head()
		if err != nil {
			return err
		}

		headCommit = r.Hash().String()
		hclog.L().Debug(fmt.Sprintf("Got head commit: %s", headCommit))
	}

	// skip commit if already processed
	{
		lastSuccessfulCommit, err := storage.Get(ctx, storageKeyLastSuccessfulCommit)
		if err != nil {
			return err
		}

		hclog.L().Debug(fmt.Sprintf("Last successful commit: %s", lastSuccessfulCommit))

		if string(lastSuccessfulCommit.Value) == headCommit {
			hclog.L().Debug(fmt.Sprintf("Head commit not changed: skipping"))
			return nil
		}
	}

	// verify head commit pgp signatures
	{
		// TODO: Check that current commit is a descendant of the last successful one

		requiredNumberOfVerifiedSignaturesOnCommit, err := getRequiredConfigurationFieldFunc(fieldNameRequiredNumberOfVerifiedSignaturesOnCommit)
		if err != nil {
			return err
		}

		trustedGpgPublicKeyNameList, err := storage.List(ctx, storageKeyPrefixTrustedGPGPublicKey)
		if err != nil {
			return err
		}

		var trustedGpgPublicKeys []string
		for _, name := range trustedGpgPublicKeyNameList {
			key, err := storage.Get(ctx, storageKeyPrefixTrustedGPGPublicKey+name)
			if err != nil {
				return err
			}

			trustedGpgPublicKeys = append(trustedGpgPublicKeys, string(key.Value))
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headCommit, trustedGpgPublicKeys, requiredNumberOfVerifiedSignaturesOnCommit.(int)); err != nil {
			return err
		}

		hclog.L().Debug(fmt.Sprintf("Verified %d commit signatures", requiredNumberOfVerifiedSignaturesOnCommit))
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

		buildTimeout, err := getRequiredConfigurationFieldFunc(fieldNameTaskTimeout)
		if err != nil {
			return err
		}

		d, err := time.ParseDuration(buildTimeout.(string))
		if err != nil {
			return err
		}

		ctxWithTimeout, ctxCancelFunc := context.WithTimeout(ctx, d)
		defer ctxCancelFunc()

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

		hclog.L().Debug(fmt.Sprintf("Running command %q in the base image %q", command, dockerImage))

		response, err := cli.ImageBuild(ctxWithTimeout, contextReader, types.ImageBuildOptions{
			NoCache:     true,
			Remove:      true,
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

		hclog.L().Debug(fmt.Sprintf("Running command %q in the base image %q DONE", command, dockerImage))
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

	hclog.L().Debug(fmt.Sprintf("Unmarshalled json: %s", entry.Value))

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

	hclog.L().Debug(fmt.Sprintf("Unexpected configuration, no path has matched pathPatternConfigure=%q", pathPatternConfigure))

	panic("runtime error")
}
