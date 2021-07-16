package kafka

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

func SelfRestoreMessage(txn *memdb.Txn, objType string, data []byte) (handled bool, err error) {
	var handler func(*memdb.Txn, interface{}) error

	switch objType {
	case model.JWTConfigType:
		handler = model.HandleRestoreConfig
	case model.JWTStateType:
		handler = model.HandleRestoreState
	default:
		return false, nil
	}

	var o interface{}
	err = json.Unmarshal(data, o)
	if err != nil {
		return false, err
	}

	err = handler(txn, o)
	if err != nil {
		return false, err
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
