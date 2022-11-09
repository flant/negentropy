package kafka

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func SelfRestoreMessage(txn io.Txn, msg sharedkafka.MsgDecoded) (handled bool, err error) {
	switch msg.Type {
	case model.JWTConfigType:
		err := model.HandleRestoreConfig(txn, msg.Data)
		if err != nil {
			return false, fmt.Errorf("handling type=%s, raw=%s: %w", msg.Type, string(msg.Data), err)
		}
	case model.JWTStateType:
		err := model.HandleRestoreState(txn, msg.Data)
		if err != nil {
			return false, fmt.Errorf("handling type=%s, raw=%s: %w", msg.Type, string(msg.Data), err)
		}
	case model.JWKSType: // TODO - REMOVE
		// TODO should this message be here or not?
		// what to do after this message?
	default:
		return false, nil
	}
	return true, nil
}

func WriteInSelfQueue(objType string) (handled bool) {
	switch objType {
	case model.JWTStateType, model.JWTConfigType:
		return true
	}

	return false
}
