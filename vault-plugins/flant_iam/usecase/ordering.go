package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type SubjectsFetcher struct {
	db      *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	entries []model.SubjectNotation
}

func NewSubjectsFetcher(db *io.MemoryStoreTxn, entries []model.SubjectNotation) *SubjectsFetcher {
	return &SubjectsFetcher{
		db:      db,
		entries: entries,
	}
}

func (snl *SubjectsFetcher) Fetch() (*model.Subjects, error) {
	result := &model.Subjects{
		ServiceAccounts: make([]model.ServiceAccountUUID, 0),
		Users:           make([]model.UserUUID, 0),
		Groups:          make([]model.GroupUUID, 0),
	}

	seen := map[string]struct{}{}

	for _, subj := range snl.entries {
		switch subj.Type {
		case model.ServiceAccountType:
			sa, err := model.NewServiceAccountRepository(snl.db).GetByID(subj.ID)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[subj.ID]; ok {
				continue
			}
			seen[subj.ID] = struct{}{}
			result.ServiceAccounts = append(result.ServiceAccounts, sa.UUID)

		case model.UserType:
			u, err := model.NewUserRepository(snl.db).GetByID(subj.ID)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[subj.ID]; ok {
				continue
			}
			seen[subj.ID] = struct{}{}
			result.Users = append(result.Users, u.UUID)

		case model.GroupType:
			g, err := model.NewGroupRepository(snl.db).GetByID(subj.ID)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[subj.ID]; ok {
				continue
			}
			seen[subj.ID] = struct{}{}
			result.Groups = append(result.Groups, g.UUID)
		}
	}

	return result, nil
}
