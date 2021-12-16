package usecase

import (
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func ListAvailableUnSafeTenants(txn *io.MemoryStoreTxn, acceptedTenants map[iam.TenantUUID]struct{}) ([]iam.Tenant, error) {
	tenants, err := usecase.Tenants(txn, consts.OriginAUTH).List(false)
	if err != nil {
		return nil, err
	}

	result := make([]iam.Tenant, 0, len(tenants))

	for _, tenant := range tenants {
		if _, accepted := acceptedTenants[tenant.UUID]; accepted {
			result = append(result, *tenant)
		}
	}
	return result, nil
}
