package usecase

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type MembersFetcher struct {
	serviceAccountRepo RawGetter
	userRepo           RawGetter
	groupRepo          RawGetter
}

func NewMembersFetcher(db *io.MemoryStoreTxn) *MembersFetcher {
	return &MembersFetcher{
		serviceAccountRepo: iam_repo.NewServiceAccountRepository(db),
		userRepo:           iam_repo.NewUserRepository(db),
		groupRepo:          iam_repo.NewGroupRepository(db),
	}
}

func (f *MembersFetcher) Fetch(members []model.MemberNotation) (*model.Members, error) {
	result := &model.Members{
		ServiceAccounts: make([]model.ServiceAccountUUID, 0),
		Users:           make([]model.UserUUID, 0),
		Groups:          make([]model.GroupUUID, 0),
	}

	seen := map[string]struct{}{}

	for _, subj := range members {
		repo, err := f.chooseRepo(subj.Type)
		if err != nil {
			return nil, err
		}

		raw, err := repo.GetRawByID(subj.UUID)
		if err != nil {
			return nil, err
		}
		m := raw.(iam_repo.Model)
		id := m.ObjId()
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}

		f.append(result, subj.Type, id)
	}

	return result, nil
}

func (f *MembersFetcher) append(result *model.Members, memberType, id string) {
	switch memberType {
	case model.ServiceAccountType:
		result.ServiceAccounts = append(result.ServiceAccounts, id)
	case model.UserType:
		result.Users = append(result.Users, id)
	case model.GroupType:
		result.Groups = append(result.Groups, id)
	}
}

func (f *MembersFetcher) chooseRepo(memberType string) (RawGetter, error) {
	var repo RawGetter

	switch memberType {
	case model.ServiceAccountType:
		repo = f.serviceAccountRepo
	case model.UserType:
		repo = f.userRepo
	case model.GroupType:
		repo = f.groupRepo
	default:
		return nil, fmt.Errorf("unsupported member type %q", memberType)
	}

	return repo, nil
}

type RawGetter interface {
	GetRawByID(string) (interface{}, error)
}
