package model

import (
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const TeammateType = "teammate" // also, memdb schema name

type FullTeammate struct {
	iam_model.User

	TeamUUID        TeamUUID   `json:"team_uuid"`
	RoleAtTeam      RoleAtTeam `json:"role_at_team"`
	GitlabAccount   string     `json:"gitlab.com"`
	GithubAccount   string     `json:"github.com"`
	TelegramAccount string     `json:"telegram"`
	HabrAccount     string     `json:"habr.com"`
}

type Teammate struct {
	memdb.ArchiveMark
	UserUUID        iam_model.UserUUID `json:"user_uuid"`
	TeamUUID        TeamUUID           `json:"team_uuid"`
	Version         string             `json:"resource_version"`
	RoleAtTeam      RoleAtTeam         `json:"role_at_team"`
	GitlabAccount   string             `json:"gitlab.com"`
	GithubAccount   string             `json:"github.com"`
	TelegramAccount string             `json:"telegram"`
	HabrAccount     string             `json:"habr.com"`
}

func (u *Teammate) ObjType() string {
	return TeammateType
}

func (u *Teammate) ObjId() string {
	return u.UserUUID
}

func (f *FullTeammate) ExtractTeammate() *Teammate {
	if f == nil {
		return nil
	}
	return &Teammate{
		ArchiveMark:     f.ArchiveMark,
		UserUUID:        f.UUID,
		TeamUUID:        f.TeamUUID,
		RoleAtTeam:      f.RoleAtTeam,
		Version:         f.Version,
		GitlabAccount:   f.GitlabAccount,
		GithubAccount:   f.GithubAccount,
		TelegramAccount: f.TelegramAccount,
		HabrAccount:     f.HabrAccount,
	}
}
