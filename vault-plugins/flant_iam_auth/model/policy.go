package model

import (
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const PolicyType = "policy" // also, memdb schema name

type Policy struct {
	memdb.ArchiveMark

	Name               PolicyName     `json:"name"` // ID
	Rego               string         `json:"rego"`
	Roles              []iam.RoleName `json:"roles"`
	ClaimSchema        string         `json:"claim_schema"`
	AllowedAuthMethods []string       `json:"allowed_auth_methods"`
}

func (p *Policy) ObjType() string {
	return PolicyType
}

func (p *Policy) ObjId() string {
	return p.Name
}
