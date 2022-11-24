package kafka

import (
	"encoding/json"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

func SelfRestoreMessage(txn io.Txn, msg io.MsgDecoded) (handled bool, err error) {
	// no deleted messages here
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
	case model.JWKSType:
		if len(msg.Data) == 0 {
			// TODO think about how to process tombstone
			// now it is a stub
			return true, nil
		}
		err := HandleRestoreOwnJwks(txn, msg.Data)
		if err != nil {
			return false, fmt.Errorf("handling type=%s, raw=%s: %w", msg.Type, string(msg.Data), err)
		}
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

func HandleRestoreOwnJwks(db io.Txn, data []byte) error {
	entry := &model.JWKS{}
	err := json.Unmarshal(data, entry)
	if err != nil {
		return fmt.Errorf("parsing: %q: %w", string(data), err)
	}

	return db.Insert(model.JWKSType, entry)
}
