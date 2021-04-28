package flant_gitops

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	goGit "github.com/go-git/go-git/v5"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
)

func (b *backend) periodic(context.Context, *logical.Request) error {
	return nil
}

func (b *backend) periodicStep(ctx context.Context, req *logical.Request) error {
	fields, err := b.getConfiguration(ctx, req)
	if err != nil {
		return err
	}

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
		repoUrl, err := getRequiredConfigurationFieldFunc(fieldGitRepoUrlName)
		if err != nil {
			return err
		}

		branchName, err := getRequiredConfigurationFieldFunc(fieldGitBranchName)
		if err != nil {
			return err
		}

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
	}

	// skip commit if already processed
	{
		lastSuccessfulCommit, err := req.Storage.Get(ctx, storageEntryLastSuccessfulCommitKey)
		if err != nil {
			return err
		}

		if string(lastSuccessfulCommit.Value) == headCommit {
			return nil
		}
	}

	// verify head commit pgp signatures
	{
		var trustedGpgPublicKeys []string
		trustedGpgPublicKeysString, err := getRequiredConfigurationFieldFunc(fieldTrustedGpgPublicKeysName)
		if err != nil {
			return err
		}

		// TODO: parse it
		_ = trustedGpgPublicKeysString

		requiredNumberOfVerifiedSignaturesOnCommit, err := getRequiredConfigurationFieldFunc(fieldRequiredNumberOfVerifiedSignaturesOnCommitName)
		if err != nil {
			return err
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headCommit, trustedGpgPublicKeys, requiredNumberOfVerifiedSignaturesOnCommit.(int)); err != nil {
			return err
		}
	}

	// run docker build with service dockerfile and context
	{
		buildDockerImage, err := getRequiredConfigurationFieldFunc(fieldBuildDockerImageName)
		if err != nil {
			return err
		}

		buildCommand, err := getRequiredConfigurationFieldFunc(fieldBuildCommandName)
		if err != nil {
			return err
		}

		// TODO: buildTimeout

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

				if err := docker.GenerateAndAddDockerfileToTar(tw, serviceDockerfilePath, serviceDirInContext, buildDockerImage.(string), []string{buildCommand.(string)}, false); err != nil {
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

		response, err := cli.ImageBuild(ctx, contextReader, types.ImageBuildOptions{
			Dockerfile:  serviceDockerfilePath,
			PullParent:  true,
			NoCache:     true,
			Remove:      true,
			ForceRemove: true,
			Version:     types.BuilderV1,
		})
		if err != nil {
			return fmt.Errorf("unable to run docker image build: %s", err)
		}

		if err := docker.DisplayFromImageBuildResponse(os.Stdout, response); err != nil {
			return err
		}
	}

	return nil
}
