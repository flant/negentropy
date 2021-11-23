package backentutils

import "github.com/hashicorp/vault/sdk/logical"

const loadingKey = "FLANT_PLUGIN_LOADING_KEY"

// IsLoading is used to recognize was passed value to config by vault
// note: need patches into vault, which will pass "true" if backend Factory is called from `startbackend` and "false" otherwise
// example (both should be placed before call bplugin.NewBackend):
// insert at original L106 vault/builtin/plugin/backend.go :
// b.config.Config["FLANT_PLUGIN_LOADING_KEY"] =  "true"
// insert at original L52 vault/builtin/plugin/backend.go original:
// conf.Config["FLANT_PLUGIN_LOADING_KEY"] = "false"
// if unmodified vault is used, expected to got "", so check for != false
func IsLoading(config *logical.BackendConfig) bool {
	if v := config.Config[loadingKey]; v != "false" {
		return true
	}
	return false
}
