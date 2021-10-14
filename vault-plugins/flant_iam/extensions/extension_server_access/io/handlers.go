package io

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
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
