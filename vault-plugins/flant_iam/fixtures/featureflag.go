package fixtures

func RandomFeatureFlagCreatePayload() map[string]interface{} {
	return map[string]interface{}{
		"name": "name_" + RandomStr(),
	}
}
