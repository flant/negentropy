package iam

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ModelHandler interface {
	HandleAuthSource(user *model.AuthSource) error
	HandleEntity(entity *model.Entity) error
	HandleEntityAlias(entity *model.EntityAlias) error
}

func HandleNewMessageSelfSource(txn *io.MemoryStoreTxn, handler ModelHandler, objType string, data []byte) error {
	var inputObject interface{}
	var entityHandler func() error

	switch objType {
	case model.AuthSourceType:
		source := &model.AuthSource{}
		inputObject = source
		// dont call here because we need unmarshal and add object in mem storage before handler
		entityHandler = func() error {
			return handler.HandleAuthSource(source)
		}
	case model.EntityType:
		entity := &model.Entity{}
		inputObject = entity
		// dont call here because we need unmarshal and add object in mem storage before handler
		entityHandler = func() error {
			return handler.HandleEntity(entity)
		}
	case model.EntityAliasType:
		entityAlias := &model.EntityAlias{}
		inputObject = entityAlias
		// dont call here because we need unmarshal and add object in mem storage before handler
		entityHandler = func() error {
			return handler.HandleEntityAlias(entityAlias)
		}
	default:
		return nil
	}

	// only unmarshal this object in mem storage
	// because we read here from self storage
	err := json.Unmarshal(data, inputObject)
	if err != nil {
		return err
	}

	if entityHandler != nil {
		return entityHandler()
	}

	return nil
}

func HandleRestoreMessagesSelfSource(txn *memdb.Txn, objType string, data []byte) error {
	var inputObject interface{}
	var table string

	// only write to mem storage
	switch objType {
	case model.AuthSourceType:
		table = model.AuthSourceType
		source := &model.AuthSource{}
		inputObject = source
	case model.AuthMethodType:
		table = model.AuthSourceType
		inputObject = &model.AuthMethod{}
	case model.EntityType:
		inputObject = &model.Entity{}
		table = model.EntityType
	case model.EntityAliasType:
		inputObject = &model.EntityAlias{}
		table = model.EntityAliasType
	case model.JWTIssueTypeType:
		inputObject = &model.JWTIssueType{}
		table = model.JWTIssueTypeType

	default:
		return nil
	}

	err := json.Unmarshal(data, inputObject)
	if err != nil {
		return err
	}

	err = txn.Insert(table, inputObject)
	if err != nil {
		return err
	}

	return nil
}
