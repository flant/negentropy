package util

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func GenerateTmpGitRepo(t *testing.T, repoName string) string {
	testRepoDir, err := ioutil.TempDir("", repoName)
	if err != nil {
		t.Fatalf("error creating tmp dir for test repo: %s", err)
	}

	ExecGitCommand(t, testRepoDir, "init")

	return testRepoDir
}

func ExecGitCommand(t *testing.T, repoDir string, args ...string) string {
	gitArgs := append([]string{"-c", "user.email=flant_gitops", "-c", "user.name=flant_gitops", "-C", repoDir}, args...)
	cmd := exec.Command("git", gitArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git command '%s' failure: %s:\n%s\n", strings.Join(append([]string{"git"}, gitArgs...), " "), err, output)
	}

	return string(output)
}

func WriteFileIntoDir(t *testing.T, path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		t.Fatalf("error creating dir %s: %s", filepath.Dir(path), err)
	}

	if err := ioutil.WriteFile(path, data, os.ModePerm); err != nil {
		t.Fatalf("error writing %s: %s", path, err)
	}
}
