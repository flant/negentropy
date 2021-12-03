package usecase

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type RoleBindingApprovalService struct {
	repo *iam_repo.RoleBindingApprovalRepository

	approverFetcher *MembersFetcher
}

func RoleBindingApprovals(db *io.MemoryStoreTxn) *RoleBindingApprovalService {
	return &RoleBindingApprovalService{
		repo:            iam_repo.NewRoleBindingApprovalRepository(db),
		approverFetcher: NewMembersFetcher(db),
	}
}

func (s RoleBindingApprovalService) GetByID(id model.RoleBindingApprovalUUID) (*model.RoleBindingApproval, error) {
	return s.repo.GetByID(id)
}

func (s RoleBindingApprovalService) List(rbID model.RoleBindingUUID,
	showArchived bool) ([]*model.RoleBindingApproval, error) {
	return s.repo.List(rbID, showArchived)
}

func (s RoleBindingApprovalService) Create(rba *model.RoleBindingApproval) error {
	if rba.Version != "" {
		return consts.ErrBadVersion
	}
	subj, err := s.approverFetcher.Fetch(rba.Approvers)
	if err != nil {
		return fmt.Errorf("RoleBindingApprovalService.Create:%w", err)
	}
	rba.Groups = subj.Groups
	rba.ServiceAccounts = subj.ServiceAccounts
	rba.Users = subj.Users
	if rba.UUID == "" {
		rba.UUID = uuid.New()
	}
	return s.repo.UpdateOrCreate(rba)
}

func (s RoleBindingApprovalService) Update(rba *model.RoleBindingApproval) error {
	if rba.UUID == "" {
		return fmt.Errorf("%w: uuid should be passed", consts.ErrInavlidArg)
	}
	subj, err := s.approverFetcher.Fetch(rba.Approvers)
	if err != nil {
		return fmt.Errorf("RoleBindingApprovalService.Update:%w", err)
	}
	rba.Groups = subj.Groups
	rba.ServiceAccounts = subj.ServiceAccounts
	rba.Users = subj.Users
	return s.repo.UpdateOrCreate(rba)
}

func (s RoleBindingApprovalService) Delete(id model.RoleBindingApprovalUUID) error {
	return s.repo.Delete(id, memdb.NewArchiveMark())
}
