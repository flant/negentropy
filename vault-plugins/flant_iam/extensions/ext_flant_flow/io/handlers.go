package io

import (
	"encoding/json"
	"fmt"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

// HandleFlantFlowObjects try to handle kafka messages as flant_flow objects
func HandleFlantFlowObjects(txn *memdb.Txn, objType string, data []byte) (handled bool, err error) {
	var object interface{}
	switch objType {
	case ext_model.TeamType:
		object = &ext_model.Team{}
	case ext_model.TeammateType:
		object = &ext_model.Teammate{}
	case ext_model.ContactType:
		object = &ext_model.Contact{}
	case ext_model.ServicePackType:
		object = &ext_model.ServicePack{}
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
