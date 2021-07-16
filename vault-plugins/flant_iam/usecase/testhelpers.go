package usecase

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func dieOnErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		panic(err)
	}
}

func checkDeepEqual(t *testing.T, expected, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("\nExpected:\n%#v\nGot:\n%#v\n", expected, got)
	}
}

func runFixtures(t *testing.T, fixtures ...func(t *testing.T, store *io.MemoryStore)) *io.MemoryStore {
	schema, err := model.GetSchema()
	dieOnErr(t, err)
	store, err := io.NewMemoryStore(schema, nil)
	dieOnErr(t, err)
	for _, fixture := range fixtures {
		fixture(t, store)
	}
	return store
}

func toMemberNotation(m model.Model) model.MemberNotation {
	return model.MemberNotation{
		Type: m.ObjType(),
		UUID: m.ObjId(),
	}
}

func toMemberNotations(ms ...model.Model) []model.MemberNotation {
	sns := make([]model.MemberNotation, 0)
	for _, m := range ms {
		sns = append(sns, toMemberNotation(m))
	}
	return sns
}

func makeMemberNotations(memberType string, uuids []string) []model.MemberNotation {
	validTypes := map[string]struct{}{model.ServiceAccountType: {}, model.UserType: {}, model.GroupType: {}}
	if _, valid := validTypes[memberType]; !valid {
		panic(fmt.Errorf("member_type %s is invalid", memberType))
	}
	result := make([]model.MemberNotation, len(uuids))
	for i := range uuids {
		result[i] = model.MemberNotation{
			Type: memberType,
			UUID: uuids[i],
		}
	}
	return result
}

func appendMembers(membersGroups ...[]model.MemberNotation) []model.MemberNotation {
	result := []model.MemberNotation{}
	for i := range membersGroups {
		result = append(result, membersGroups[i]...)
	}
	return result
}
