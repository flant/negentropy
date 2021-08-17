package ssh_session

// func to specify a restrictions for a valid SSHSessionRunParams
type sshSessionRunParamsChecker func(params SSHSessionRunParams) error

// func to specify valid SSHSessionRunParams
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
