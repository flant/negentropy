package ssh_session

// func to specify a restrictions for a valid SSHSessionRunParams
// return error if Params invalid, it finishes and fails validation
type sshSessionRunParamsChecker func(params SSHSessionRunParams) error

// func to specify valid SSHSessionRunParams
// return true if Params valid, it finishes validation with success
type sshSessionRunParamsAllower func(params SSHSessionRunParams) bool

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
	checkers = []sshSessionRunParamsChecker{} // TODO write and fill
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
