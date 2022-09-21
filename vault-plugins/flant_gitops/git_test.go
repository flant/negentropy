package flant_gitops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	goGit "github.com/go-git/go-git/v5"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

func repoWithTwoCommits(t *testing.T) (string, *goGit.Repository, []gitCommitHash) {
	testGitRepoDir := util.GenerateTmpGitRepo(t, "flant_gitops_test_repo")
	defer os.RemoveAll(testGitRepoDir)

	util.ExecGitCommand(t, testGitRepoDir, "checkout", "-b", "main")
	util.WriteFileIntoDir(t, filepath.Join(testGitRepoDir, "data"), []byte("OUTPUT1\n"))
	util.ExecGitCommand(t, testGitRepoDir, "add", ".")
	util.ExecGitCommand(t, testGitRepoDir, "commit", "-m", "one")
	commit1 := strings.TrimSpace(util.ExecGitCommand(t, testGitRepoDir, "rev-parse", "HEAD"))

	//fmt.Printf("Current commit in test repo %s: %s\n", testGitRepoDir, commit1)

	util.WriteFileIntoDir(t, filepath.Join(testGitRepoDir, "data"), []byte("OUTPUT2\n"))
	util.ExecGitCommand(t, testGitRepoDir, "add", ".")
	util.ExecGitCommand(t, testGitRepoDir, "commit", "-m", "two")
	commit2 := strings.TrimSpace(util.ExecGitCommand(t, testGitRepoDir, "rev-parse", "HEAD"))

	//fmt.Printf("Current commit in test repo %s: %s\n", testGitRepoDir, commit2)

	var cloneOptions trdlGit.CloneOptions
	{
		cloneOptions.BranchName = "main"
		cloneOptions.RecurseSubmodules = goGit.DefaultSubmoduleRecursionDepth
	}
	gitRepo, err := trdlGit.CloneInMemory(testGitRepoDir, cloneOptions)
	require.NoError(t, err)

	return testGitRepoDir, gitRepo, []gitCommitHash{commit1, commit2}
}

func Test_git_collectAllCommits(t *testing.T) {
	//_, gitRepo, expectedCommits := repoWithTwoCommits(t)
	//
	//gotCommits, err := collectCommitsFrom(gitRepo, "")
	//
	//require.NoError(t, err)
	//for i := range gotCommits {
	//	require.Equal(t, expectedCommits[i], gotCommits[i].Hash.String())
	//}
}

func Test_git_collectOnlyLastCommit(t *testing.T) {
	//_, gitRepo, expectedCommits := repoWithTwoCommits(t)
	//
	//gotCommits, err := collectCommitsFrom(gitRepo, expectedCommits[0])
	//
	//require.NoError(t, err)
	//require.Equal(t, 1, len(gotCommits))
	//require.Equal(t, expectedCommits[1], gotCommits[0].Hash.String())
}
