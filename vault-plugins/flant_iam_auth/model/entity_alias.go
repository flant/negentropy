package model

const EntityAliasType = "entity_alias" // also, memdb schema name

type EntityAlias struct {
	UUID       string `json:"uuid"`        // ID
	UserId     string `json:"user_id"`     // user is user or sa or multipass
	Name       string `json:"name"`        // source name. by it vault look alias for user
	SourceName string `json:"source_name"` //
}

func (p *EntityAlias) ObjType() string {
	return EntityAliasType
}

func (p *EntityAlias) ObjId() string {
	return p.UUID
}
