package flant_gitops

import (
	"fmt"
	"time"
)

const (
	storageKeyConfiguration              = "configuration"
	storageKeyConfigurationGitCredential = "configuration_git_credential"
)

type gitCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type configuration struct {
	GitRepoUrl                                 string `json:"git_repo_url"`
	GitBranch                                  string `json:"git_branch"`
	GitPollPeriod                              string `json:"git_poll_period"`
	RequiredNumberOfVerifiedSignaturesOnCommit int    `json:"required_number_of_verified_signatures_on_commit"`
	InitialLastSuccessfulCommit                string `json:"initial_last_successful_commit"`
	DockerImage                                string `json:"docker_image"`
	Command                                    string `json:"command"`
}

func (c *configuration) GetGitPollPeroid() time.Duration {
	d, err := time.ParseDuration(c.GitPollPeriod)
	if err != nil {
		panic(fmt.Sprintf("invalid git poll period duration: %s", err))
	}
	return d
}
