package flant_gitops

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
	GitBranchName                              string `json:"git_branch_name"`
	GitPollPeriod                              string `json:"git_poll_period"`
	RequiredNumberOfVerifiedSignaturesOnCommit string `json:"required_number_of_verified_signatures_on_commit"`
	InitialLastSuccessfulCommit                string `json:"initial_last_successful_commit"`
	DockerImage                                string `json:"docker_image"`
	Command                                    string `json:"command"`
}
