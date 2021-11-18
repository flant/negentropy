package io

import (
	"encoding/json"
	"fmt"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

// HandleServerAccessObjects try to handle kafka messages as ServerAccess objects
func HandleServerAccessObjects(txn *memdb.Txn, objType string, data []byte) (handled bool, err error) {
	var object interface{}
	switch objType {
	case ext_model.ServerType:
		object = &ext_model.Server{}
	default:
		return false, nil
	}
	err = json.Unmarshal(data, object)
	if err != nil {
		return false, fmt.Errorf("parsing: %w", err)
	}
	err = txn.Insert(objType, object)
	if err != nil {
		return false, fmt.Errorf("saving: %w", err)
	}
	return true, nil
}
