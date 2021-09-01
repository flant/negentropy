package usecase

import (
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func ListAvailableSafeTenants(txn *io.MemoryStoreTxn, acceptedTenants map[iam.TenantUUID]struct{}) ([]model.SafeTenant, error) {
	tenants, err := usecase.Tenants(txn).List(false)
	if err != nil {
		return nil, err
	}

	result := make([]model.SafeTenant, 0, len(tenants))

	for _, tenant := range tenants {
		if _, accepted := acceptedTenants[tenant.UUID]; accepted {
			res := model.SafeTenant{
				UUID:    tenant.UUID,
				Version: tenant.Version,
			}
			result = append(result, res)
		}
	}
	return result, nil
}
