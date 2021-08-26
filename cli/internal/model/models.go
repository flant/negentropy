package model

import (
	"fmt"

	"github.com/flant/negentropy/cli/internal/consts"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type ServerFilter struct {
	AllTenants        bool
	AllProjects       bool
	AllServers        bool
	TenantIdentifier  string
	ProjectIdentifier string
	LabelSelector     string
	ServerIdentifiers []string
}

type ServerList struct {
	Tenants  map[iam.TenantUUID]iam.Tenant
	Projects map[iam.ProjectUUID]iam.Project
	Servers  map[ext.ServerUUID]ext.Server
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
	if params.AllTenants || params.TenantIdentifier != "" {
		return nil
	}
	return fmt.Errorf("needs %s or %s flag", consts.AllTenantsFlagName, consts.TenantFlagName)
}

func needProjectsIfTenant(params ServerFilter) error {
	if !params.AllTenants {
		if params.AllProjects || params.ProjectIdentifier != "" {
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
	if ((params.AllServers || len(params.ServerIdentifiers) > 0) && params.LabelSelector != "") ||
		((!params.AllServers || len(params.ServerIdentifiers) == 0) && params.LabelSelector == "") {
		return fmt.Errorf("needs one of : %s, or servers identifiers, or %s", consts.AllServersFlagName,
			consts.LabelsFlagName)
	}
	return nil
}
