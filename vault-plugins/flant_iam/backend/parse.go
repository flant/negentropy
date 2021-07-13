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

func parseSubjects(rawList interface{}) ([]model.SubjectNotation, error) {
	subjects := make([]model.SubjectNotation, 0)

	if rawList == nil {
		return subjects, nil
	}

	rawSubjects, ok := rawList.([]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot parse subjects list")
	}

	for _, raw := range rawSubjects {
		s, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot parse subject %v", raw)
		}
		subj := model.SubjectNotation{
			Type: s["type"].(string),
			ID:   s["id"].(string),
		}
		subjects = append(subjects, subj)
	}

	return subjects, nil
}

func parseBoundRoles(rawList interface{}) ([]model.BoundRole, error) {
	boundRoles := make([]model.BoundRole, 0)

	if rawList == nil {
		return boundRoles, nil
	}

	rawSubjects, ok := rawList.([]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot parse subjects list")
	}

	for _, raw := range rawSubjects {
		s, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot parse subject %v", raw)
		}
		subj := model.BoundRole{
			Name:    s["name"].(string),
			Options: s["options"].(map[string]interface{}),
		}
		boundRoles = append(boundRoles, subj)
	}

	return boundRoles, nil
}
