package model

const EntityType = "entity" // also, memdb schema name

type Entity struct {
	UUID   string `json:"uuid"` // ID
	Name   string `json:"name"` // Identifier
	UserId string `json:"user_id"`
}

func (p *Entity) ObjType() string {
	return EntityType
}

func (p *Entity) ObjId() string {
	return p.UUID
}
