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

type ProjectUUIDWithIdentifier struct {
	UUID       string `json:"uuid"`
	Identifier string `json:"identifier"`
}

type ProjectMapper interface {
	Denormalize([]model.ProjectUUID) ([]ProjectUUIDWithIdentifier, error)
}

type projectMapper struct {
	projectRepo *iam_repo.ProjectRepository
}

func (m projectMapper) denormalizeOne(projectUUID model.ProjectUUID) (*ProjectUUIDWithIdentifier, error) {
	project, err := m.projectRepo.GetByID(projectUUID)
	if err != nil {
		return nil, fmt.Errorf("catching project by UUID {%s}: %w", projectUUID, err)
	}
	return &ProjectUUIDWithIdentifier{
		UUID:       projectUUID,
		Identifier: project.Identifier,
	}, nil
}

func (m projectMapper) Denormalize(projects []model.ProjectUUID) ([]ProjectUUIDWithIdentifier, error) {
	result := make([]ProjectUUIDWithIdentifier, 0, len(projects))
	for _, projectUUID := range projects {
		projectUUIDWithIdentifier, err := m.denormalizeOne(projectUUID)
		if err != nil {
			return nil, err
		}
		result = append(result, *projectUUIDWithIdentifier)
	}
	return result, nil
}

func NewProjectMapper(db *io.MemoryStoreTxn) ProjectMapper {
	return projectMapper{
		projectRepo: iam_repo.NewProjectRepository(db),
	}
}
