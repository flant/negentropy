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

func toSubjectNotation(m model.Model) model.SubjectNotation {
	return model.SubjectNotation{
		Type: m.ObjType(),
		ID:   m.ObjId(),
	}
}

func toSubjectNotations(ms ...model.Model) []model.SubjectNotation {
	sns := make([]model.SubjectNotation, 0)
	for _, m := range ms {
		sns = append(sns, toSubjectNotation(m))
	}
	return sns
}

func makeSubjectNotations(subjectType string, uuids []string) []model.SubjectNotation {
	validTypes := map[string]struct{}{model.ServiceAccountType: {}, model.UserType: {}, model.GroupType: {}}
	if _, valid := validTypes[subjectType]; !valid {
		panic(fmt.Errorf("subject_type %s is invalid", subjectType))
	}
	result := make([]model.SubjectNotation, len(uuids))
	for i := range uuids {
		result[i] = model.SubjectNotation{
			Type: subjectType,
			ID:   uuids[i],
		}
	}
	return result
}

func appendSubjects(subjectsGroups ...[]model.SubjectNotation) []model.SubjectNotation {
	result := []model.SubjectNotation{}
	for i := range subjectsGroups {
		result = append(result, subjectsGroups[i]...)
	}
	return result
}
