package model

import (
	"crypto/rsa"
)

const ReplicaType = "plugin_replica" // also, memdb schema name

type Replica struct {
	Name      ReplicaName    `json:"name"`
	TopicType string         `json:"type"`
	PublicKey *rsa.PublicKey `json:"replica_key"`
}

func (r Replica) ObjType() string {
	return ReplicaType
}

func (r Replica) ObjId() string {
	return r.Name
}
