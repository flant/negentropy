package usecase

import (
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func ListAvailableSafeProjects(txn *io.MemoryStoreTxn, tenantID string, acceptedProjects map[iam.ProjectUUID]struct{}) ([]model.SafeProject, error) {
	projects, err := usecase.Projects(txn).List(tenantID, false)
	if err != nil {
		return nil, err
	}

	result := make([]model.SafeProject, 0, len(projects))

	for _, project := range projects {
		_, projectAcccepted := acceptedProjects[project.UUID]
		if projectAcccepted {
			res := model.SafeProject{
				UUID:       project.UUID,
				TenantUUID: project.TenantUUID,
				Version:    project.Version,
			}
			result = append(result, res)
		}
	}
	return result, nil
}
