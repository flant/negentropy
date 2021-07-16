package backend

import (
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func getCreationID(expectID bool, data *framework.FieldData) string {
	var id string

	if expectID {
		// for privileged access
		id = data.Get("uuid").(string)
	}

	if id == "" {
		id = uuid.New()
	}

	return id
}

func parseMembers(rawList interface{}) ([]model.MemberNotation, error) {
	members := make([]model.MemberNotation, 0)

	if rawList == nil {
		return members, nil
	}

	rawMembers, ok := rawList.([]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot parse members list")
	}

	for _, raw := range rawMembers {
		s, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot parse member %v", raw)
		}
		typ, ok := s["type"].(string)
		if !ok {
			return nil, fmt.Errorf("cannot parse type in %v", raw)
		}
		id, ok := s["id"].(string)
		if !ok {
			return nil, fmt.Errorf("cannot parse id in %v", raw)
		}

		subj := model.MemberNotation{
			Type: typ,
			ID:   id,
		}
		members = append(members, subj)
	}

	return members, nil
}

func parseBoundRoles(rawList interface{}) ([]model.BoundRole, error) {
	boundRoles := make([]model.BoundRole, 0)

	if rawList == nil {
		return boundRoles, nil
	}

	rawBoundRoles, ok := rawList.([]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot parse roles list")
	}

	for _, raw := range rawBoundRoles {
		s, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot parse role %v", raw)
		}

		name, ok := s["name"].(string)
		if !ok {
			return nil, fmt.Errorf("cannot parse name in %v", raw)
		}
		options, ok := s["options"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot parse options in %v", raw)
		}
		br := model.BoundRole{
			Name:    name,
			Options: options,
		}
		boundRoles = append(boundRoles, br)
	}

	return boundRoles, nil
}
