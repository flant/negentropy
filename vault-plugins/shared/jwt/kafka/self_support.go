package kafka

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func SelfRestoreMessage(txn *memdb.Txn, objType string, data []byte) (handled bool, err error) {
	switch objType {
	case model.JWTConfigType:
		err := model.HandleRestoreConfig(txn, data)
		if err != nil {
			return false, fmt.Errorf("handling type=%s, raw=%s: %w", objType, string(data), err)
		}
	case model.JWTStateType:
		err := model.HandleRestoreState(txn, data)
		if err != nil {
			return false, fmt.Errorf("handling type=%s, raw=%s: %w", objType, string(data), err)
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
