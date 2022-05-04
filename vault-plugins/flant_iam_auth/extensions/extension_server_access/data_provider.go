package extension_server_access

import (
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type DataProvider struct {
	serverRepository *repo.ServerRepository
}

// GetExtensionData returns all servers at tenant/project
func (p DataProvider) GetExtensionData(_ model.Subject,
	claim model.RoleClaim) (map[string]interface{}, error) {
	serversPTR, err := p.serverRepository.List(claim.TenantUUID, claim.ProjectUUID, false)
	if err != nil {
		return nil, err
	}
	servers := make([]ext.Server, 0, len(serversPTR))
	for _, s := range serversPTR {
		servers = append(servers, *s)
	}
	return map[string]interface{}{"servers": servers}, nil
}

func NewDataProvider(tx *io.MemoryStoreTxn) DataProvider {
	return DataProvider{serverRepository: repo.NewServerRepository(tx)}
}
