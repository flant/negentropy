package model

import (
	"fmt"
	"testing"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	groupUUID1 = "00000000-0001-0000-0000-000000000000"
	groupUUID2 = "00000000-0002-0000-0000-000000000000"
	groupUUID3 = "00000000-0003-0000-0000-000000000000"
	groupUUID4 = "00000000-0004-0000-0000-000000000000"
	groupUUID5 = "00000000-0005-0000-0000-000000000000"
)

func makeSubjectNotations(subjectType string, uuids []string) []SubjectNotation {
	validTypes := map[string]struct{}{ServiceAccountType: {}, UserType: {}, GroupType: {}}
	if _, valid := validTypes[subjectType]; !valid {
		panic(fmt.Errorf("subject_type %s is invalid", subjectType))
	}
	result := make([]SubjectNotation, len(uuids))
	for i := range uuids {
		result[i] = SubjectNotation{
			Type: subjectType,
			ID:   uuids[i],
		}
	}
	return result
}

func appendSubjects(subjectsGroups ...[]SubjectNotation) []SubjectNotation {
	result := []SubjectNotation{}
	for i := range subjectsGroups {
		result = append(result, subjectsGroups[i]...)
	}
	return result
}

var (
	group1 = Group{
		UUID:            groupUUID1,
		TenantUUID:      tenantUUID1,
		Users:           []string{userUUID2, userUUID3},
		Groups:          []string{groupUUID3},
		ServiceAccounts: []string{serviceAccountUUID1},
		Origin:          OriginIAM,
	}
	group2 = Group{
		UUID:       groupUUID2,
		TenantUUID: tenantUUID1,
		Users:      []string{userUUID1, userUUID3},
		Origin:     OriginIAM,
	}
	group3 = Group{
		UUID:            groupUUID3,
		TenantUUID:      tenantUUID2,
		Users:           []string{userUUID3, userUUID4},
		ServiceAccounts: []string{serviceAccountUUID1},
		Origin:          OriginIAM,
	}
	group4 = Group{
		UUID:            groupUUID4,
		TenantUUID:      tenantUUID1,
		Users:           []string{userUUID2, userUUID3},
		Groups:          []string{groupUUID2, groupUUID3},
		ServiceAccounts: []string{serviceAccountUUID2, serviceAccountUUID3},
		Origin:          OriginIAM,
	}
	group5 = Group{
		UUID:       groupUUID5,
		TenantUUID: tenantUUID1,
		Groups:     []string{groupUUID2, groupUUID1},
		Origin:     OriginIAM,
	}
)

func createGroups(t *testing.T, repo *GroupRepository, groups ...Group) {
	for _, group := range groups {
		tmp := group
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func groupFixture(t *testing.T, store *io.MemoryStore) {
	gs := []Group{group2, group3, group4, group1, group5}
	for i := range gs {
		gs[i].Subjects = appendSubjects(makeSubjectNotations(UserType, gs[i].Users),
			makeSubjectNotations(ServiceAccountType, gs[i].ServiceAccounts),
			makeSubjectNotations(GroupType, gs[i].Groups))
	}
	tx := store.Txn(true)
	repo := NewGroupRepository(tx)
	createGroups(t, repo, gs...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_GroupDbSchema(t *testing.T) {
	schema := GroupSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("group schema is invalid: %v", err)
	}
}

func Test_ListGroups(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := NewGroupRepository(tx)

	ids, err := repo.List(tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, []string{groupUUID1, groupUUID2, groupUUID4, groupUUID5}, ids)
}

func Test_GetByID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := NewGroupRepository(tx)

	group, err := repo.GetByID(groupUUID1)

	dieOnErr(t, err)
	group1.Subjects = appendSubjects(makeSubjectNotations(UserType, group1.Users),
		makeSubjectNotations(ServiceAccountType, group1.ServiceAccounts),
		makeSubjectNotations(GroupType, group1.Groups))
	group.Version = group1.Version
	group.FullIdentifier = group1.FullIdentifier
	checkDeepEqual(t, &group1, group)
}

func Test_findDirectParentGroupsByUserUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := NewGroupRepository(tx)

	ids, err := repo.findDirectParentGroupsByUserUUID(tenantUUID1, userUUID3)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}, groupUUID2: {}, groupUUID4: {}}, ids)
}

func Test_findDirectParentGroupsByServiceAccountUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := NewGroupRepository(tx)

	ids, err := repo.findDirectParentGroupsByServiceAccountUUID(tenantUUID1, serviceAccountUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}}, ids)
}

func Test_findDirectParentGroupsByGroupUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := NewGroupRepository(tx)

	ids, err := repo.findDirectParentGroupsByGroupUUID(tenantUUID1, groupUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID5: {}}, ids)
}

func Test_FindAllParentGroupsForUserUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := NewGroupRepository(tx)

	ids, err := repo.FindAllParentGroupsForUserUUID(tenantUUID1, userUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID2: {}, groupUUID4: {}, groupUUID5: {}}, ids)
}

func Test_FindAllParentGroupsForServiceAccountUUID(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture).Txn(true)
	repo := NewGroupRepository(tx)

	ids, err := repo.FindAllParentGroupsForServiceAccountUUID(tenantUUID1, serviceAccountUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{groupUUID1: {}, groupUUID5: {}}, ids)
}
