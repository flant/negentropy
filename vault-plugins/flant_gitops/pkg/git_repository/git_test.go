package git_repository

import (
	"testing"

	goGit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

func repoWithTwoCommits(t *testing.T) (*goGit.Repository, []gitCommitHash) {
	testGitRepo, err := util.NewTestGitRepo("flant_gitops_test_repo")
	defer testGitRepo.Clean()
	require.NoError(t, err)

	err = testGitRepo.WriteFileIntoRepoAndCommit("data", []byte("OUTPUT1\n"), "one")
	require.NoError(t, err)

	err = testGitRepo.WriteFileIntoRepoAndCommit("data", []byte("OUTPUT2\n"), "two")
	require.NoError(t, err)

	gitRepo, err := testGitRepo.GetClonedInMemoryGitRepo()
	require.NoError(t, err)

	return gitRepo, testGitRepo.CommitHashes
}

func Test_git_collectAllCommits(t *testing.T) {
	gitRepo, expectedCommits := repoWithTwoCommits(t)

	gotCommits, err := collectCommitsFromSomeTillHead(gitRepo, "")

	require.NoError(t, err)
	for i := range gotCommits {
		require.Equal(t, expectedCommits[i], *gotCommits[i])
	}
}

func Test_git_collectOnlyLastCommit(t *testing.T) {
	gitRepo, expectedCommits := repoWithTwoCommits(t)

	gotCommits, err := collectCommitsFromSomeTillHead(gitRepo, expectedCommits[0])

	require.NoError(t, err)
	require.Equal(t, 1, len(gotCommits))
	require.Equal(t, expectedCommits[1], *gotCommits[0])
}
