package model

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	ServerType = "server" // also, memdb schema name
)

func ServerSchema() *memdb.DBSchema {
	var serverIdentifierMultiIndexer []memdb.Indexer

	tenantUUIDIndex := &memdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, tenantUUIDIndex)

	projectUUIDIndex := &memdb.StringFieldIndex{
		Field:     "ProjectUUID",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, projectUUIDIndex)

	serverIdentifierIndex := &memdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	serverIdentifierMultiIndexer = append(serverIdentifierMultiIndexer, serverIdentifierIndex)

	var tenantProjectMultiIndexer []memdb.Indexer
	tenantProjectMultiIndexer = append(tenantProjectMultiIndexer, tenantUUIDIndex)
	tenantProjectMultiIndexer = append(tenantProjectMultiIndexer, projectUUIDIndex)

	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ServerType: {
				Name: ServerType,
				Indexes: map[string]*memdb.IndexSchema{
					model.PK: {
						Name:   model.PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					model.TenantForeignPK: {
						Name: model.TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					model.ProjectForeignPK: {
						Name: model.ProjectForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "ProjectUUID",
							Lowercase: true,
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &memdb.CompoundIndex{
							Indexes: serverIdentifierMultiIndexer,
						},
					},
					"tenant_project": {
						Name: "tenant_project",
						Indexer: &memdb.CompoundIndex{
							Indexes: tenantProjectMultiIndexer},
					},
				},
			},
		},
	}
}

type Server struct {
	UUID           string            `json:"uuid"` // ID
	TenantUUID     string            `json:"tenant_uuid"`
	ProjectUUID    string            `json:"project_uuid"`
	Version        string            `json:"resource_version"`
	Identifier     string            `json:"identifier"`
	FullIdentifier string            `json:"full_identifier"` // calculated <identifier>@<tenant_identifier>
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`
}

func (u *Server) ObjType() string {
	return ServerType
}

func (u *Server) ObjId() string {
	return u.UUID
}

func (u *Server) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(u)
}

func (u *Server) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, u)
	return err
}
