package model

import "github.com/flant/negentropy/vault-plugins/shared/io"

type Subjects struct {
	ServiceAccounts []ServiceAccountUUID
	Users           []UserUUID
	Groups          []GroupUUID
}

type SubjectNotation struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type SubjectsFetcher struct {
	db      *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	entries []SubjectNotation
}

func NewSubjectsFetcher(db *io.MemoryStoreTxn, entries []SubjectNotation) *SubjectsFetcher {
	return &SubjectsFetcher{
		db:      db,
		entries: entries,
	}
}

func (snl *SubjectsFetcher) Fetch() (*Subjects, error) {
	result := &Subjects{
		ServiceAccounts: make([]ServiceAccountUUID, 0),
		Users:           make([]UserUUID, 0),
		Groups:          make([]GroupUUID, 0),
	}

	seen := map[string]struct{}{}

	for _, subj := range snl.entries {
		switch subj.Type {
		case ServiceAccountType:
			sa, err := NewServiceAccountRepository(snl.db).GetByID(subj.ID)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[subj.ID]; ok {
				continue
			}
			seen[subj.ID] = struct{}{}
			result.ServiceAccounts = append(result.ServiceAccounts, sa.UUID)

		case UserType:
			u, err := NewUserRepository(snl.db).GetByID(subj.ID)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[subj.ID]; ok {
				continue
			}
			seen[subj.ID] = struct{}{}
			result.Users = append(result.Users, u.UUID)

		case GroupType:
			g, err := NewGroupRepository(snl.db).GetByID(subj.ID)
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
