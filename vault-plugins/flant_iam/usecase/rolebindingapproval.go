package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleBindingApprovalService struct {
	repo *iam_repo.RoleBindingApprovalRepository
}

func RoleBindingApprovals(db *io.MemoryStoreTxn) *RoleBindingApprovalService {
	return &RoleBindingApprovalService{
		repo: iam_repo.NewRoleBindingApprovalRepository(db),
	}
}

func (s RoleBindingApprovalService) GetByID(id model.RoleBindingApprovalUUID) (*model.RoleBindingApproval, error) {
	return s.repo.GetByID(id)
}

func (s RoleBindingApprovalService) List(rbID model.RoleBindingUUID,
	showArchived bool) ([]*model.RoleBindingApproval, error) {
	return s.repo.List(rbID, showArchived)
}

func (s RoleBindingApprovalService) Update(roleBindingApproval *model.RoleBindingApproval) error {
	return s.repo.UpdateOrCreate(roleBindingApproval)
}

func (s RoleBindingApprovalService) Delete(id model.RoleBindingApprovalUUID) error {
	archivingTimestamp, archivingHash := ArchivingLabel()
	return s.repo.Delete(id, archivingTimestamp, archivingHash)
}
