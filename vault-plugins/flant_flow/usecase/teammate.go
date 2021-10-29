package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_flow/iam_clients"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// TODO add work with IAM

type TeammateService struct {
	repo       *repo.TeammateRepository
	userClient iam_clients.UserClient
}

func Teammates(db *io.MemoryStoreTxn, userClient iam_clients.UserClient) *TeammateService {
	return &TeammateService{
		repo:       repo.NewTeammateRepository(db),
		userClient: userClient,
	}
}

func (s *TeammateService) Create(t *model.Teammate) error {
	// TODO check role exists
	// check team exists
	// check role corresponds team type
	// TODO sync with IAM

	t.Version = repo.NewResourceVersion()
	return s.repo.Create(t)
}

func (s *TeammateService) Update(updated *model.Teammate) error {
	// TODO check role exists
	// check team exists
	// check role corresponds team type
	// TODO sync with IAM
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

func (s *TeammateService) Delete(id iam_model.UserUUID) error {
	// TODO sync with IAM
	archivingTimestamp, archivingHash := ArchivingLabel()
	return s.repo.Delete(id, archivingTimestamp, archivingHash)
}

func (s *TeammateService) GetByID(id iam_model.UserUUID) (*model.Teammate, error) {
	return s.repo.GetByID(id)
}

func (s *TeammateService) List(showArchived bool) ([]*model.Teammate, error) {
	return s.repo.List(showArchived)
}

func (s *TeammateService) Restore(id iam_model.UserUUID) (*model.Teammate, error) {
	// TODO sync with IAM
	return s.repo.Restore(id)
}
