package usecase

import (
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func ListAvailableProjects(txn *io.MemoryStoreTxn, tenantID string, acceptedProjects map[iam.ProjectUUID]struct{}) ([]model.Project, error) {
	projects, err := usecase.Projects(txn, consts.OriginAUTH).List(tenantID, false)
	if err != nil {
		return nil, err
	}

	result := make([]model.Project, 0, len(projects))

	for _, project := range projects {
		_, projectAcccepted := acceptedProjects[project.UUID]
		if projectAcccepted {
			res := model.Project{
				UUID:       project.UUID,
				TenantUUID: project.TenantUUID,
				Version:    project.Version,
				Identifier: project.Identifier,
			}
			result = append(result, res)
		}
	}
	return result, nil
}
