package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
)

type TestGitRepo struct {
	// full path to repo
	RepoDir      string
	CommitHashes []string
}

func NewTestGitRepo(repoName string) (*TestGitRepo, error) {
	testRepoDir, err := ioutil.TempDir("", repoName)
	if err != nil {
		return nil, fmt.Errorf("error creating tmp dir for test repo: %w", err)
	}

	_, err = ExecGitCommand(testRepoDir, "init")
	if err != nil {
		return nil, err
	}

	return &TestGitRepo{
		RepoDir:      testRepoDir,
		CommitHashes: []string{},
	}, nil
}

func ExecGitCommand(repoDir string, args ...string) (string, error) {
	gitArgs := append([]string{"-c", "user.email=flant_gitops", "-c", "user.name=flant_gitops", "-C", repoDir}, args...)
	cmd := exec.Command("git", gitArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command '%s' failure: %s:\n%s\n", strings.Join(append([]string{"git"}, gitArgs...), " "), err, output)
	}

	return string(output), nil
}

func (r *TestGitRepo) WriteFileIntoRepoAndCommit(relativePath string, fileData []byte, commitMessage string) error {
	path := filepath.Join(r.RepoDir, relativePath)

	if err := ioutil.WriteFile(path, fileData, os.ModePerm); err != nil {
		return fmt.Errorf("error writing %s: %s", path, err)
	}

	if _, err := ExecGitCommand(r.RepoDir, "add", "."); err != nil {
		return fmt.Errorf("error adding %s: %s", path, err)
	}

	if _, err := ExecGitCommand(r.RepoDir, "commit", "-m", commitMessage); err != nil {
		return fmt.Errorf("error commiting: %w", err)
	}

	out, err := ExecGitCommand(r.RepoDir, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("error collecting commit hash: %w", err)
	}
	commit := strings.TrimSpace(out)
	r.CommitHashes = append(r.CommitHashes, commit)
	return nil
}

func (r *TestGitRepo) GetClonedInMemoryGitRepo() (*git.Repository, error) {
	if _, err := ExecGitCommand(r.RepoDir, "checkout", "-b", "main"); err != nil {
		return nil, fmt.Errorf("error checkout: %w", err)
	}

	var cloneOptions trdlGit.CloneOptions
	{
		cloneOptions.BranchName = "main"
		cloneOptions.RecurseSubmodules = git.DefaultSubmoduleRecursionDepth
	}

	gitRepo, err := trdlGit.CloneInMemory(r.RepoDir, cloneOptions)
	if err != nil {
		return nil, err
	}
	return gitRepo, nil
}

func (r *TestGitRepo) Clean() {
	if r != nil && r.RepoDir != "" {
		os.RemoveAll(r.RepoDir) // nolint:errcheck
	}
}
