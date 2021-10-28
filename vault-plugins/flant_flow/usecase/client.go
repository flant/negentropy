package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_flow/iam_clients"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	repo "github.com/flant/negentropy/vault-plugins/flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// TODO add work with IAM

type ClientService struct {
	repo         *repo.ClientRepository
	tenantClient iam_clients.TenantClient
	// subtenants
	childrenDeleters []DeleterByParent
}

func Clients(db *io.MemoryStoreTxn, tenantClient iam_clients.TenantClient) *ClientService {
	return &ClientService{
		repo:             repo.NewClientRepository(db),
		tenantClient:     tenantClient,
		childrenDeleters: []DeleterByParent{},
	}
}

func (s *ClientService) Create(t *model.Client) error {
	t.Version = repo.NewResourceVersion()
	return s.repo.Create(t)
}

func (s *ClientService) Update(updated *model.Client) error {
	stored, err := s.repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}

	// Validate

	if stored.Version != updated.Version {
		return model.ErrBadVersion
	}
	updated.Version = repo.NewResourceVersion()

	// Update

	return s.repo.Create(updated)
}

func (s *ClientService) Delete(id model.ClientUUID) error {
	archivingTimestamp, archivingHash := ArchivingLabel()
	if err := deleteChildren(id, s.childrenDeleters, archivingTimestamp, archivingHash); err != nil {
		return err
	}
	return s.repo.Delete(id, archivingTimestamp, archivingHash)
}

func (s *ClientService) GetByID(id model.ClientUUID) (*model.Client, error) {
	return s.repo.GetByID(id)
}

func (s *ClientService) List(showArchived bool) ([]*model.Client, error) {
	return s.repo.List(showArchived)
}

func (s *ClientService) Restore(id model.ClientUUID, fullRestore bool) (*model.Client, error) {
	if fullRestore {
		// TODO check if full restore available
		// TODO FullRestore
		return s.repo.Restore(id)
	}
	// TODO Short Restore
	return s.repo.Restore(id)
}
