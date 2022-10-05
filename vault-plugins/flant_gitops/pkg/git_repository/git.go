package git_repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
)

type gitCommitHash = string

type gitService struct {
	ctx     context.Context
	storage logical.Storage
	logger  hclog.Logger
}

func GitService(ctx context.Context, storage logical.Storage, logger hclog.Logger) gitService {
	return gitService{
		ctx:     ctx,
		storage: storage,
		logger:  logger,
	}
}

// CheckForNewCommit returns new commit hashes satisfied given rules
// 1. check defined by cfg repo&branch
// 2. collect all commits after spicified
// 3. returns first commit signed with specified amount of PGP, after last processed
func (g gitService) CheckForNewCommitFrom(edgeCommit gitCommitHash) (*gitCommitHash, error) {
	config, err := GetConfig(g.ctx, g.storage, g.logger)
	if err != nil {
		return nil, err
	}

	gitRepo, newCommits, err := g.getNewCommits(config, edgeCommit)
	if err != nil {
		return nil, fmt.Errorf("getting new commits: %w", err)
	}
	if len(newCommits) == 0 {
		g.logger.Debug("no new commits: nothing to check ")
		return nil, nil
	}

	return g.getFirstSignedCommit(gitRepo, newCommits, config.RequiredNumberOfVerifiedSignaturesOnCommit)
}

// getNewCommits returns new commits after "lastProcessed"
func (g gitService) getNewCommits(config *Configuration, edgeCommit gitCommitHash) (*goGit.Repository, []*gitCommitHash, error) {
	lastProcessedCommit := g.EdgeCommit(config.InitialLastSuccessfulCommit, edgeCommit)

	// clone git repository and get head commit
	g.logger.Debug(fmt.Sprintf("Cloning git repo %q branch %q", config.GitRepoUrl, config.GitBranch))
	gitRepo, headCommit, err := g.cloneGit(config.GitRepoUrl, config.GitBranch)
	// skip if already processed
	if lastProcessedCommit == headCommit {
		g.logger.Debug("Head commit not changed: no new commits")
		return nil, nil, nil
	}

	newCommits, err := collectCommitsFromSomeTillHead(gitRepo, lastProcessedCommit)

	return gitRepo, newCommits, err
}

// cloneGit clones specified repo, checkout specified branch and return head commit of branch
func (g gitService) cloneGit(GitRepoUrl, GitBranch string) (*goGit.Repository, gitCommitHash, error) {
	gitCredentials, err := trdlGit.GetGitCredential(g.ctx, g.storage)
	if err != nil {
		return nil, "", fmt.Errorf("unable to get Git credentials Configuration: %s", err)
	}

	var cloneOptions trdlGit.CloneOptions
	{
		cloneOptions.BranchName = GitBranch
		// cloneOptions.RecurseSubmodules = goGit.DefaultSubmoduleRecursionDepth //

		if gitCredentials != nil && gitCredentials.Username != "" && gitCredentials.Password != "" {
			cloneOptions.Auth = &http.BasicAuth{
				Username: gitCredentials.Username,
				Password: gitCredentials.Password,
			}
		}
	}

	var gitRepo *goGit.Repository
	if gitRepo, err = trdlGit.CloneInMemory(GitRepoUrl, cloneOptions); err != nil {
		return nil, "", err
	}

	r, err := gitRepo.Head()
	if err != nil {
		return nil, "", err
	}
	headCommit := r.Hash().String()
	g.logger.Debug(fmt.Sprintf("Got head commit: %s", headCommit))
	return gitRepo, headCommit, nil
}

// EdgeCommit returns last proceeded commit
func (g gitService) EdgeCommit(initialLastSuccessfulCommit gitCommitHash, edgeCommit gitCommitHash) gitCommitHash {
	result := edgeCommit
	if edgeCommit == "" {
		result = initialLastSuccessfulCommit
	}

	g.logger.Debug(fmt.Sprintf("Edge commit: %s", result))
	return result
}

// collectCommitsFrom collects commits from commit (not includes it) and check is they are descendant
func collectCommitsFromSomeTillHead(gitRepo *goGit.Repository, edgeCommit gitCommitHash) ([]*gitCommitHash, error) {
	ref, err := gitRepo.Head()
	if err != nil {
		return nil, err
	}
	headCommit := ref.Hash().String()

	if edgeCommit != "" {
		isAncestor, err := trdlGit.IsAncestor(gitRepo, edgeCommit, headCommit)
		if err != nil {
			return nil, err
		}

		if !isAncestor {
			return nil, fmt.Errorf("git commit %q is not descendant of the commit %q", headCommit, edgeCommit)
		}
	}

	commit, err := gitRepo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	commitIter, err := gitRepo.Log(&goGit.LogOptions{From: commit.Hash})
	if err != nil {
		return nil, err
	}

	var result []*gitCommitHash
	for c, err := commitIter.Next(); err == nil; c, err = commitIter.Next() {
		if c.Hash.String() == edgeCommit {
			break
		}
		commitHash := c.Hash.String()
		result = append([]*gitCommitHash{&commitHash}, result...) // need reverse flow of commits
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return result, nil
}

// getFirstSignedCommit returns first commit signed with specified amount of signs
func (g gitService) getFirstSignedCommit(gitRepo *goGit.Repository, commits []*gitCommitHash,
	requiredNumberOfVerifiedSignaturesOnCommit int) (*gitCommitHash, error) {
	trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(g.ctx, g.storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get trusted public keys: %s", err)
	}

	for _, c := range commits {
		err = trdlGit.VerifyCommitSignatures(gitRepo, *c, trustedPGPPublicKeys, requiredNumberOfVerifiedSignaturesOnCommit)
		if err != nil {
			g.logger.Debug(fmt.Sprintf("checking commit signing: %s: %s", *c, err.Error()))
		}
		return c, nil
	}
	return nil, nil
}

func GetConfig(ctx context.Context, storage logical.Storage, logger hclog.Logger) (*Configuration, error) {
	config, err := getConfiguration(ctx, storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get Configuration: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("Configuration not set")
	}

	cfgData, _ := json.MarshalIndent(config, "", "  ") // nolint:errcheck
	logger.Debug(fmt.Sprintf("Got Configuration:\n%s", string(cfgData)))

	return config, nil
}
