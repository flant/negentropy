package vault

import "github.com/hashicorp/vault/api"

func SecretDataSetString(secret *api.Secret, key string, value string) {
	if secret.Data == nil {
		secret.Data = make(map[string]interface{})
	}
	secret.Data[key] = value
}

func SecretDataGetString(secret *api.Secret, key string, defaults ...string) string {
	var val = secret.Data[key].(string)
	if secret.Data != nil && val != "" {
		return val
	}
	for _, def := range defaults {
		if def != "" {
			return def
		}
	}
	return ""
}
