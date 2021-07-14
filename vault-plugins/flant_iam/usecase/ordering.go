package usecase

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type SubjectsFetcher struct {
	serviceAccountRepo RawGetter
	userRepo           RawGetter
	groupRepo          RawGetter
}

func NewSubjectsFetcher(db *io.MemoryStoreTxn) *SubjectsFetcher {
	return &SubjectsFetcher{
		serviceAccountRepo: model.NewServiceAccountRepository(db),
		userRepo:           model.NewUserRepository(db),
		groupRepo:          model.NewGroupRepository(db),
	}
}

func (f *SubjectsFetcher) Fetch(subjects []model.SubjectNotation) (*model.Subjects, error) {
	result := &model.Subjects{
		ServiceAccounts: make([]model.ServiceAccountUUID, 0),
		Users:           make([]model.UserUUID, 0),
		Groups:          make([]model.GroupUUID, 0),
	}

	seen := map[string]struct{}{}

	for _, subj := range subjects {
		repo, err := f.chooseRepo(subj.Type)
		if err != nil {
			return nil, err
		}

		raw, err := repo.GetRawByID(subj.ID)
		if err != nil {
			return nil, err
		}
		m := raw.(model.Model)
		id := m.ObjId()
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}

		f.append(result, subj.Type, id)
	}

	return result, nil
}

func (f *SubjectsFetcher) append(result *model.Subjects, subjectType, id string) {
	switch subjectType {
	case model.ServiceAccountType:
		result.ServiceAccounts = append(result.ServiceAccounts, id)
	case model.UserType:
		result.Users = append(result.Users, id)
	case model.GroupType:
		result.Groups = append(result.Groups, id)
	}
}

func (f *SubjectsFetcher) chooseRepo(subjectType string) (RawGetter, error) {
	var repo RawGetter

	switch subjectType {
	case model.ServiceAccountType:
		repo = f.serviceAccountRepo
	case model.UserType:
		repo = f.userRepo
	case model.GroupType:
		repo = f.groupRepo
	default:
		return nil, fmt.Errorf("unsupported subject type %q", subjectType)
	}

	return repo, nil
}

type RawGetter interface {
	GetRawByID(string) (interface{}, error)
}
