package self

import (
	"encoding/json"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type ModelHandler interface {
	HandleAuthSource(user *model.AuthSource) error
	HandleEntity(entity *model.Entity) error
	HandleEntityAlias(entity *model.EntityAlias) error

	DeletedAuthSource(uuid string) error
	DeletedEntity(uuid string) error
	DeletedEntityAlias(uuid string) error
}

func HandleNewMessageSelfSource(txn *io.MemoryStoreTxn, handler ModelHandler, msg *sharedkafka.MsgDecoded) error {
	isDelete := msg.IsDeleted()

	var inputObject interface{}
	var entityHandler func() error

	objId := msg.ID

	switch msg.Type {
	case model.AuthSourceType:
		source := &model.AuthSource{}
		inputObject = source
		// dont call here because we need unmarshal and add object in mem storage before handler
		if isDelete {
			entityHandler = func() error {
				return handler.DeletedAuthSource(objId)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleAuthSource(source)
			}
		}

	case model.EntityType:
		entity := &model.Entity{}
		inputObject = entity
		// dont call here because we need unmarshal and add object in mem storage before handler
		if isDelete {
			entityHandler = func() error {
				return handler.DeletedEntity(objId)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleEntity(entity)
			}
		}

	case model.EntityAliasType:
		entityAlias := &model.EntityAlias{}
		inputObject = entityAlias
		// dont call here because we need unmarshal and add object in mem storage before handler
		if isDelete {
			entityHandler = func() error {
				return handler.DeletedEntityAlias(objId)
			}
		} else {
			entityHandler = func() error {
				return handler.HandleEntityAlias(entityAlias)
			}
		}

	case model.AuthMethodType, model.MethodTypeJWT:
		// don't need handle
		return nil

	default:
		return nil
	}

	if !isDelete {
		// only unmarshal this object in mem storage
		// because we read here from self storage
		err := json.Unmarshal(msg.Data, inputObject)
		if err != nil {
			return err
		}
	}

	if entityHandler != nil {
		return entityHandler()
	}

	return nil
}
