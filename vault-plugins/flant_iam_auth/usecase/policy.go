package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type PolicyService struct {
	repo *repo.PolicyRepository
}

func Policies(db *io.MemoryStoreTxn) *PolicyService {
	return &PolicyService{
		repo: repo.NewPolicyRepository(db),
	}
}

func (s *PolicyService) Create(policy *model.Policy) error {
	_, err := s.repo.GetByID(policy.Name)
	if !errors.Is(err, consts.ErrNotFound) {
		return fmt.Errorf("%w: %s", consts.ErrAlreadyExists, policy.Name)
	}
	if err != nil && !errors.Is(err, consts.ErrNotFound) {
		return err
	}
	return s.repo.Create(policy)
}

func (s *PolicyService) Update(updated *model.Policy) error {
	stored, err := s.repo.GetByID(updated.Name)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	// Validate
	return s.repo.Update(updated)
}

func (s *PolicyService) Delete(name model.PolicyName) error {
	_, err := s.repo.GetByID(name)
	if err != nil {
		return err
	}
	return s.repo.Delete(name, memdb.NewArchiveMark())
}

func (s *PolicyService) Erase(name model.PolicyName) error {
	policy, err := s.repo.GetByID(name)
	if err != nil {
		return err
	}
	if policy.NotArchived() {
		return consts.ErrIsNotArchived
	}
	return s.repo.Erase(name)
}

func (s *PolicyService) GetByID(name model.PolicyName) (*model.Policy, error) {
	return s.repo.GetByID(name)
}

func (s *PolicyService) List(showArchived bool) ([]model.PolicyName, error) {
	return s.repo.ListIDs(showArchived)
}
