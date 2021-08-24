package ssh_session

import (
	"fmt"

	"github.com/flant/negentropy/cli/internals/consts"
)

// func to specify a restrictions for a valid SSHSessionRunParams
// return error if Params invalid, it finishes and fails validation
type sshSessionRunParamsChecker func(SSHSessionRunParams) error

// func to specify valid SSHSessionRunParams
// return true if Params valid, it finishes validation with success
type sshSessionRunParamsAllower func(SSHSessionRunParams) bool

type SSHSessionRunParams struct {
	AllTenants  bool
	AllProjects bool
	Tenant      string
	Project     string
	Labels      []string
	Args        []string
}

var (
	allowers = []sshSessionRunParamsAllower{} // TODO write and fill
	checkers = []sshSessionRunParamsChecker{
		needTenants,
		needProjects,
	} // TODO write and fill
)

func (p SSHSessionRunParams) Validate() error {
	for _, allower := range allowers {
		if allower(p) {
			return nil
		}
	}
	for _, checker := range checkers {
		if err := checker(p); err != nil {
			return err
		}
	}
	return nil
}

func needTenants(params SSHSessionRunParams) error {
	if params.AllTenants || params.Tenant != "" {
		return nil
	}
	return fmt.Errorf("needs %s or %s flag", consts.AllTenantsFlagName, consts.TenantFlagName)
}

func needProjects(params SSHSessionRunParams) error {
	if params.AllProjects || params.Project != "" {
		return nil
	}
	return fmt.Errorf("needs %s or %s flag", consts.AllProjectsFlagName, consts.ProjectFlagName)
}
