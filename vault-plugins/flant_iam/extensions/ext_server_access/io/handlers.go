package io

import (
	"encoding/json"
	"fmt"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// HandleServerAccessObjects try to handle kafka messages as ServerAccess objects
func HandleServerAccessObjects(txn io.Txn, msg io.MsgDecoded) (handled bool, err error) {
	handled, err = io.HandleTombStone(txn, msg)
	if handled || err != nil {
		return handled, err
	}

	var object interface{}
	switch msg.Type {
	case ext_model.ServerType:
		object = &ext_model.Server{}
	default:
		return false, nil
	}
	err = json.Unmarshal(msg.Data, object)
	if err != nil {
		return false, fmt.Errorf("parsing: %w", err)
	}
	err = txn.Insert(msg.Type, object)
	if err != nil {
		return false, fmt.Errorf("saving: %w", err)
	}
	return true, nil
}
