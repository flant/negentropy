package usecase

import (
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// iam_usecase.Tenants implements every thing
func Clients(db *io.MemoryStoreTxn) *iam_usecase.TenantService {
	return iam_usecase.Tenants(db, consts.OriginFlantFlow)
}

// TODO do we need filter not flant flow tanants at /LIST ?
