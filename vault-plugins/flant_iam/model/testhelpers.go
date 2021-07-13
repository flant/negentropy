package model

import (
	"fmt"
	"reflect"
	"testing"

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
	schema, err := mergeSchema()
	dieOnErr(t, err)
	store, err := io.NewMemoryStore(schema, nil)
	dieOnErr(t, err)
	for _, fixture := range fixtures {
		fixture(t, store)
	}
	return store
}

func toSubjectNotation(m Model) SubjectNotation {
	return SubjectNotation{
		Type: m.ObjType(),
		ID:   m.ObjId(),
	}
}

func toSubjectNotations(ms ...Model) []SubjectNotation {
	sns := make([]SubjectNotation, 0)
	for _, m := range ms {
		sns = append(sns, toSubjectNotation(m))
	}
	return sns
}

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
