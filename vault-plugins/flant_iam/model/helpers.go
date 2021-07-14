package model

func stringSlice(uuidSet map[string]struct{}) []string {
	if len(uuidSet) == 0 {
		return nil
	}
	result := make([]string, 0, len(uuidSet))
	for uuid := range uuidSet {
		result = append(result, uuid)
	}
	return result
}

func stringSet(uuidSlice []string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, uuid := range uuidSlice {
		result[uuid] = struct{}{}
	}
	return result
}
