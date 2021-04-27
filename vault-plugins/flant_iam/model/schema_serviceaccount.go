package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

const (
	ServiceAccountType = "service_account" // also, memdb schema name

)

type ServiceAccountObjectType string

const (
	SaTypeGeneric ServiceAccountObjectType = "generic"
	SaTypeBuiltin ServiceAccountObjectType = "builtin"
)

func ServiceAccountSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ServiceAccountType: {
				Name: ServiceAccountType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name:   TenantForeignPK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					"version": {
						Name: "version",
						Indexer: &memdb.StringFieldIndex{
							Field: "Version",
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &memdb.StringFieldIndex{
							Field: "Identifier",
						},
					},
					"type": {
						Name: "type",
						Indexer: &memdb.StringFieldIndex{
							Field: "Type",
						},
					},
					"full_identifier": {
						Name: "full_identifier",
						Indexer: &memdb.StringFieldIndex{
							Field: "FullIdentifier",
						},
					},
					"cirds": {
						Name: "cirds",
						Indexer: &memdb.StringSliceFieldIndex{
							Field: "CIRDs",
						},
					},
					"ttl": {
						Name: "ttl",
						Indexer: &memdb.IntFieldIndex{
							Field: "TTL",
						},
					},
					"max_ttl": {
						Name: "max_ttl",
						Indexer: &memdb.IntFieldIndex{
							Field: "MaxTTL",
						},
					},
				},
			},
		},
	}
}

type ServiceAccount struct {
	UUID           string `json:"uuid"` // ID
	TenantUUID     string `json:"tenant_uuid"`
	Version        string `json:"resource_version"`
	Type           ServiceAccountObjectType
	Identifier     string        `json:"identifier"`
	FullIdentifier string        `json:"full_identifier"`
	CIDRs          []string      `json:"allowed_cidrs"`
	TTL            time.Duration `json:"token_ttl"`
	MaxTTL         time.Duration `json:"token_max_ttl"`
}

func (u *ServiceAccount) ObjType() string {
	return ServiceAccountType
}

func (u *ServiceAccount) ObjId() string {
	return u.UUID
}

func (u *ServiceAccount) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(u)
}

func (u *ServiceAccount) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, u)
	return err
}
