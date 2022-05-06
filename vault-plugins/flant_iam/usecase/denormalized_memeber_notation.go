package usecase

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type DenormalizedMemberNotation struct {
	Type           string `json:"type"`
	UUID           string `json:"uuid"`
	Identifier     string `json:"identifier"`
	FullIdentifier string `json:"full_identifier"`
}

type MemberNotationMapper interface {
	Denormalize([]model.MemberNotation) ([]DenormalizedMemberNotation, error)
}

type memberNotationMapper struct {
	serviceAccountRepo *iam_repo.ServiceAccountRepository
	userRepo           *iam_repo.UserRepository
	groupRepo          *iam_repo.GroupRepository
}

func (m memberNotationMapper) denormalizeOne(member model.MemberNotation) (*DenormalizedMemberNotation, error) {
	denormilizedMember := DenormalizedMemberNotation{
		Type: member.Type,
		UUID: member.UUID,
	}
	switch member.Type {
	case model.UserType:
		user, err := m.userRepo.GetByID(member.UUID)
		if err != nil {
			return nil, fmt.Errorf("catching member {%#v}: %w", member, err)
		}
		denormilizedMember.Identifier = user.Identifier
		denormilizedMember.FullIdentifier = user.FullIdentifier
	case model.ServiceAccountType:
		serviceAccount, err := m.serviceAccountRepo.GetByID(member.UUID)
		if err != nil {
			return nil, fmt.Errorf("catching member {%#v}: %w", member, err)
		}
		denormilizedMember.Identifier = serviceAccount.Identifier
		denormilizedMember.FullIdentifier = serviceAccount.FullIdentifier
	case model.GroupType:
		group, err := m.groupRepo.GetByID(member.UUID)
		if err != nil {
			return nil, fmt.Errorf("catching member {%#v}: %w", member, err)
		}
		denormilizedMember.Identifier = group.Identifier
		denormilizedMember.FullIdentifier = group.FullIdentifier
	default:
		return nil, fmt.Errorf("wrong type of member: %s", member.Type)
	}
	return &denormilizedMember, nil
}

func (m memberNotationMapper) Denormalize(members []model.MemberNotation) ([]DenormalizedMemberNotation, error) {
	result := make([]DenormalizedMemberNotation, 0, len(members))
	for _, member := range members {
		denormilizedMember, err := m.denormalizeOne(member)
		if err != nil {
			return nil, err
		}
		result = append(result, *denormilizedMember)
	}
	return result, nil
}

func NewMemberNotationMapper(db *io.MemoryStoreTxn) MemberNotationMapper {
	return memberNotationMapper{
		serviceAccountRepo: iam_repo.NewServiceAccountRepository(db),
		userRepo:           iam_repo.NewUserRepository(db),
		groupRepo:          iam_repo.NewGroupRepository(db),
	}
}
