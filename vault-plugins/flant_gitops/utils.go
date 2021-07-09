package flant_gitops

import "encoding/json"

// TODO: move to trdl?
// All numbers converted to float64.
func jsonStructToMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(data, &resultMap); err != nil {
		return nil, err
	}

	return resultMap, nil
}
