package model

import (
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type (
	TeamUUID        = string
	RoleAtTeam      = string
	ClientUUID      = iam_model.TenantUUID
	UnixTime        = int64
	ServicePackName = string
	ProjectUUID     = string
	ContactUUID     = string
)

// Team types
const (
	StandardTeam = "standard_team"
	DevopsTeam   = "devops_team"
)

var (
	AllowedTeamTypes = []interface{}{StandardTeam, DevopsTeam}
	TeamTypes        = map[string]struct{}{StandardTeam: {}, DevopsTeam: {}}
)

// Roles at teams
var (
	MemberRole         RoleAtTeam = "member"
	EngineerRole       RoleAtTeam = "engineer"
	ManagerRole        RoleAtTeam = "manager"
	ProjectManagerRole RoleAtTeam = "project_manager"
	TeamLeadRole       RoleAtTeam = "teamlead"

	DevopsTeamRoles = map[RoleAtTeam]struct{}{
		MemberRole: {}, EngineerRole: {}, ManagerRole: {}, ProjectManagerRole: {}, TeamLeadRole: {},
	}
	StardardTeamRoles = map[RoleAtTeam]struct{}{MemberRole: {}, ManagerRole: {}}

	AllowedRolesAtTeam = []interface{}{MemberRole, EngineerRole, ManagerRole, ProjectManagerRole, TeamLeadRole}
	TeamRoles          = map[string]map[RoleAtTeam]struct{}{
		StandardTeam: StardardTeamRoles,
		DevopsTeam:   DevopsTeamRoles,
	}
)

type LinkedGroupType = string
