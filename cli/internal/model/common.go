package model

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/flant/negentropy/cli/internal/consts"
)

type ServerFilter struct {
	AllTenants         bool
	AllProjects        bool
	AllServers         bool
	TenantIdentifiers  StringSet
	ProjectIdentifiers StringSet
	LabelSelector      string
	ServerIdentifiers  []string
}

type VaultSSHSignRequest struct {
	PublicKey       string `json:"public_key"`
	ValidPrincipals string `json:"valid_principals"`
}

// func to specify a restrictions for a valid ServerFilter
// return error if Params invalid, it finishes and fails validation
type serverFilterChecker func(ServerFilter) error

// func to specify valid ServerFilter
// return true if Params valid, it finishes validation with success
type serverFilterAllower func(ServerFilter) bool

var (
	allowers = []serverFilterAllower{} // TODO write and fill
	checkers = []serverFilterChecker{
		needTenant,
		needProjectsIfTenant,
		needOneOfAllServersFlagOrServersIdentifiers,
		needServersIdentifiersOrLabelSelector,
	} // TODO write and fill
)

func (p ServerFilter) Validate() error {
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

func needTenant(params ServerFilter) error {
	if params.AllTenants || !params.TenantIdentifiers.IsEmpty() {
		return nil
	}
	return fmt.Errorf("needs %s or %s flag", consts.AllTenantsFlagName, consts.TenantFlagName)
}

func needProjectsIfTenant(params ServerFilter) error {
	if !params.AllTenants {
		if params.AllProjects || !params.ProjectIdentifiers.IsEmpty() {
			return nil
		}
		return fmt.Errorf("needs %s or %s flag in case of tenant is defined", consts.AllProjectsFlagName,
			consts.ProjectFlagName)
	}
	return nil
}

func needOneOfAllServersFlagOrServersIdentifiers(params ServerFilter) error {
	if params.AllServers && len(params.ServerIdentifiers) > 0 {
		return fmt.Errorf("needs one of : %s or servers identifiers", consts.AllServersFlagName)
	}
	return nil
}

func needServersIdentifiersOrLabelSelector(params ServerFilter) error {
	if len(params.ServerIdentifiers) > 0 && params.LabelSelector != "" {
		return fmt.Errorf("needs one of : %s, or servers identifiers, or %s", consts.AllServersFlagName,
			consts.LabelsFlagName)
	}
	return nil
}

type StringSet map[string]struct{}

func (s *StringSet) Put(identifier string) {
	(*s)[identifier] = struct{}{}
}

func (s *StringSet) Contains(identifier string) bool {
	if _, ok := (*s)[identifier]; ok {
		return true
	}
	return false
}

func (s *StringSet) Length() int {
	return len(*s)
}

func (s *StringSet) IsEmpty() bool {
	return len(*s) == 0
}

func StringSetFromStringFlag(flags *pflag.FlagSet, flagName string) (StringSet, error) {
	s, err := flags.GetString(flagName)
	if err != nil {
		return StringSet{}, err
	}
	if s == "" {
		return StringSet{}, nil
	}
	return StringSet{s: struct{}{}}, nil
}
