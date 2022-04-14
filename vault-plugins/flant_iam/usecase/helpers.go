package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
)

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

func roleNames(chains map[model.RoleName]repo.RoleChain) []string {
	if len(chains) == 0 {
		return nil
	}
	result := make([]string, 0, len(chains))
	for roleName := range chains {
		result = append(result, roleName)
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
