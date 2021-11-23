package repo

import (
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const ClientForeignPK = "tenant_uuid"

type ClientRepository = iam_repo.TenantRepository

func NewClientRepository(txn *io.MemoryStoreTxn) *ClientRepository {
	return iam_repo.NewTenantRepository(txn)
}
