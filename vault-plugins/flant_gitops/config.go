package flant_gitops

const storageKeyConfigurationGitCredential = "configuration_git_credential"

type gitCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
